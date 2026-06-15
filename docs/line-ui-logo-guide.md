# LINE UI and Logo Guide

## 1) Prepare brand assets

- App icon (square): 1024x1024 PNG
- Hero/banner image: 1080x878 PNG (safe area centered)
- Rich menu background: use LINE rich menu size guidance (large and small templates)
- Font style: keep Thai readability first and high contrast colors

Tips:

- Keep logo simple and readable at small sizes
- Avoid tiny text in icon/logo
- Keep one primary color + one accent color

## 2) Set profile visual in LINE Official Account Manager

1. Open LINE Official Account Manager
2. Go to account settings
3. Upload profile image (logo)
4. Set display name and status message

## 3) Build chat UI with Rich Menu

1. Open Rich menu settings in Official Account Manager
2. Create menu areas matching your bot flows (for example: FAQ, นัดหมาย, คุยเจ้าหน้าที่, ฉุกเฉิน)
3. Upload background image
4. Bind each area to message/postback action
5. Publish and test on mobile

## 4) Build message cards with Flex Message Simulator

1. Open https://developers.line.biz/flex-simulator/
2. Design bubble/carousel templates
3. Export JSON
4. Put JSON in your backend payload builder (or config if your system supports it)
5. Validate text length and Thai line breaks on real device

## 5) UX checklist for healthcare bot

- First screen must include: emergency option and staff handoff
- Keep one message short: 3-5 lines for readability
- Use quick replies to reduce user typing
- Add confirmation copy for sensitive actions
- Test on both Android and iOS LINE clients

## 6) Governance checklist

- Avoid storing sensitive patient data in logo/UI assets
- Keep consistent medical disclaimer in persistent menu
- Review wording with clinical team before publish

## 7) Recommended workflow with this repository

1. Draft UI labels from your diagnosis flow
2. Map each menu tap to backend keyword intent
3. Test click-through path in a staging LINE channel
4. Release gradually and monitor fallback/error metrics
