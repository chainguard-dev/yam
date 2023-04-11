package formatted

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncoder_AutomaticConfig(t *testing.T) {
	t.Run("gracefully handles missing config file", func(t *testing.T) {
		err := os.Chdir("testdata/empty-dir")
		require.NoError(t, err)

		w := new(bytes.Buffer)

		assert.NotPanics(t, func() {
			_ = NewEncoder(w).AutomaticConfig()
		})
	})
}
