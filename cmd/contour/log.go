package main

import (
	"github.com/sirupsen/logrus"
)

// bridge verbose flag into a logrus.Level
func logruslevel(v int) (l logrus.Level) {
	if v >= 0 && v <= 5 {
		l = logrus.AllLevels[v]
	} else if v > 5 {
		l = logrus.DebugLevel
	} else {
		l = logrus.PanicLevel
	}
	return
}
