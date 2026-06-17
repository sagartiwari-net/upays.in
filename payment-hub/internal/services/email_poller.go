package services

type EmailPoller interface {
	TriggerPoll()
}
