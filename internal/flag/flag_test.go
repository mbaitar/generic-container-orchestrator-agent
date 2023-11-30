package flag

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetAndHas(t *testing.T) {
	assert.False(t, Has(IgnoreInstanceDiff), "should not have flag enabled")

	// enable flag
	Set(IgnoreInstanceDiff)
	assert.True(t, Has(IgnoreInstanceDiff), "should have flag enabled")

	// disable flag
	Clear(IgnoreInstanceDiff)
	assert.False(t, Has(IgnoreInstanceDiff), "should have cleared the flag")
}

func TestReset(t *testing.T) {
	// enable flag
	Set(IgnoreInstanceDiff)
	assert.True(t, Has(IgnoreInstanceDiff), "should have flag enabled")

	// reset all flags
	Reset()
	assert.False(t, Has(IgnoreInstanceDiff), "should have cleared the flag")
}
