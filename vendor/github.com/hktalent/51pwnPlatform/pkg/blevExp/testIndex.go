package blevExp

import (
	"time"
)

func TestIndex() {
	SaveIndexDoc("github", "xx1", map[string]interface{}{
		"xx":  44,
		"xx3": "9987",
	}, func() {

	}, nil)

	SaveIndexDoc("github", "xx2", map[string]interface{}{
		"xx":  "test good",
		"xx2": "test good",
		"xx3": "9987",
	}, func() {

	}, nil)

	SaveIndexDoc("github", "xx3", map[string]interface{}{
		"xx":    "test good 3",
		"test1": true,
		"xx2":   time.Now(),
		"xx3":   "9987",
	}, func() {

	}, nil)
}
