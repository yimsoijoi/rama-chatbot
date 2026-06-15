package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"obgynrama-chatbot/internal/observability"
	"obgynrama-chatbot/internal/usecase"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

type LineHandler struct {
	bot     *linebot.Client
	usecase *usecase.ReplyUsecase
	logger  *slog.Logger
	dedup   observability.EventDeduplicator
}

func NewLineHandler(bot *linebot.Client, uc *usecase.ReplyUsecase, logger *slog.Logger) *LineHandler {
	return NewLineHandlerWithDedup(bot, uc, logger, nil)
}

func NewLineHandlerWithDedup(bot *linebot.Client, uc *usecase.ReplyUsecase, logger *slog.Logger, dedup observability.EventDeduplicator) *LineHandler {
	if dedup == nil {
		dedup = noOpDeduplicator{}
	}
	return &LineHandler{bot: bot, usecase: uc, logger: logger, dedup: dedup}
}

func (h *LineHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	requestID := observability.RequestIDFromContext(r.Context())
	h.logger.Info("webhook_received",
		slog.String("request_id", requestID),
		slog.String("path", r.URL.Path),
		slog.Bool("signature_present", r.Header.Get("x-line-signature") != ""),
	)

	events, err := h.bot.ParseRequest(r)
	if err != nil {
		if errors.Is(err, linebot.ErrInvalidSignature) {
			h.logger.Warn("webhook_invalid_signature",
				slog.String("request_id", requestID),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error("webhook_parse_failed",
			slog.String("request_id", requestID),
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, event := range events {
		if event.WebhookEventID != "" && h.dedup.SeenOrAdd(event.WebhookEventID) {
			h.logger.Info("webhook_duplicate_skipped",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
			)
			continue
		}

		eventUserID := ""
		if event.Source != nil {
			eventUserID = event.Source.UserID
		}

		if event.DeliveryContext.IsRedelivery {
			h.logger.Info("webhook_redelivery_received",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
				slog.String("line_user_id", eventUserID),
			)
		}

		if event.Type == linebot.EventTypeUnsend {
			h.logger.Info("webhook_unsend_received",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
				slog.String("line_user_id", eventUserID),
			)
			continue
		}

		if event.Type != linebot.EventTypeMessage {
			h.logger.Info("webhook_skip_event_type",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
				slog.String("event_type", string(event.Type)),
			)
			continue
		}

		msg, ok := event.Message.(*linebot.TextMessage)
		if !ok {
			h.logger.Info("webhook_skip_non_text_message",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
				slog.String("event_type", string(event.Type)),
			)
			continue
		}

		userID := ""
		if event.Source != nil {
			userID = event.Source.UserID
		}

		result := h.usecase.BuildReply(userID, msg.Text)
		replyMsg := linebot.NewTextMessage(result.Message)
		if len(result.QuickReply) > 0 {
			items := make([]*linebot.QuickReplyButton, 0, len(result.QuickReply))
			for i, q := range result.QuickReply {
				if q == "" {
					continue
				}
				if i >= 13 {
					break // LINE quick reply max is 13 items
				}
				items = append(items, linebot.NewQuickReplyButton("", linebot.NewMessageAction(q, q)))
			}
			if len(items) > 0 {
				replyMsg.WithQuickReplies(linebot.NewQuickReplyItems(items...))
			}
		}

		replyCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		resp, err := h.bot.ReplyMessage(event.ReplyToken, replyMsg).WithContext(replyCtx).Do()
		cancel()
		if err != nil {
			h.logger.Error("webhook_reply_failed",
				slog.String("request_id", requestID),
				slog.String("webhook_event_id", event.WebhookEventID),
				slog.Bool("is_redelivery", event.DeliveryContext.IsRedelivery),
				slog.String("line_user_id", userID),
				slog.String("error", err.Error()),
			)
			continue
		}

		lineRequestID := ""
		if resp != nil {
			lineRequestID = resp.RequestID
		}
		h.logger.Info("webhook_reply_sent",
			slog.String("request_id", requestID),
			slog.String("webhook_event_id", event.WebhookEventID),
			slog.String("line_request_id", lineRequestID),
			slog.String("line_user_id", userID),
		)

		// Link per-user rich menu when DX is selected
		if result.RichMenuID != "" && userID != "" {
			linkCtx, linkCancel := context.WithTimeout(r.Context(), 5*time.Second)
			_, linkErr := h.bot.LinkUserRichMenu(userID, result.RichMenuID).WithContext(linkCtx).Do()
			linkCancel()
			if linkErr != nil {
				h.logger.Warn("rich_menu_link_failed",
					slog.String("request_id", requestID),
					slog.String("line_user_id", userID),
					slog.String("rich_menu_id", result.RichMenuID),
					slog.String("error", linkErr.Error()),
				)
			} else {
				h.logger.Info("rich_menu_linked",
					slog.String("request_id", requestID),
					slog.String("line_user_id", userID),
					slog.String("rich_menu_id", result.RichMenuID),
				)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

type noOpDeduplicator struct{}

func (noOpDeduplicator) SeenOrAdd(string) bool {
	return false
}
