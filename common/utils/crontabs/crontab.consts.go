package crontabs

import "time"

const (
	Pre30Seconds = "*/30 * * * * *"
	Pre1Minutes  = "0 */1 * * * *"
)

const (
	AHalfDay = time.Hour * 12
)
