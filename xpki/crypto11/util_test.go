package crypto11

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TokensInfo(t *testing.T) {
	slots, err := p11lib.TokensInfo()
	require.NoError(t, err)
	assert.NotNil(t, slots)
	assert.True(t, len(slots) > 0, "At least one slot must already exist")
	for _, si := range slots {
		if si.id == 0 {
			continue
		}
		if si.serial != "" {
			assert.NotEmpty(t, si.label)
		}
	}
}

func Test_GetSlotKeys(t *testing.T) {
	slots, err := p11lib.TokensInfo()
	require.NoError(t, err)
	assert.NotNil(t, slots)
	assert.True(t, len(slots) > 0, "At least one slot must already exist")
	for _, si := range slots {
		if si.id == 0 {
			continue
		}
		if si.serial != "" {
			count := 0
			err := p11lib.EnumKeys(si.id, "", func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error {
				count++
				return nil
			})
			require.NoError(t, err)
		}
	}
}
