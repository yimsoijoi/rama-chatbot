package repository

type UserDiagnosisRepository interface {
	GetDiagnosisByLineUserID(lineUserID string) (string, bool, error)
	SetDiagnosisByLineUserID(lineUserID, diagnosis string) error
}
