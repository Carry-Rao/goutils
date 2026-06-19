package api

import "time"

type Table interface {
	Ins(any, time.Duration) error
	Get(any, []string, time.Duration) (any, error)
	Set(any, []string, time.Duration) error
	Delete(any, []string, time.Duration) error
}
