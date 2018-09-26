package sdscallback

type CallbackMessage interface {
	ToJSON() ([]byte, error)
}
