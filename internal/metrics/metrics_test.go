package metrics

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEchoHandler_Get(t *testing.T) {
	delta := int64(1000)
	value := float64(1000.1)
	resJSON := `[{"id":"test1","type":"counter","delta":1000},{"id":"test2","type":"gauge","value":1000.1}]`
	resGo := []Metrics{
		{ID: "test1", MType: "counter", Delta: &delta},
		{ID: "test2", MType: "gauge", Value: &value},
	}
	ret, err := json.Marshal(resGo)
	require.NoError(t, err)
	zap.L().Info(string(ret))
	assert.Equal(t, string(ret), resJSON)
	resGo2 := make([]Metrics, 0)
	require.NoError(t, json.Unmarshal([]byte(resJSON), &resGo2))
	assert.Equal(t, *resGo[0].Delta, *resGo2[0].Delta)
	assert.Equal(t, *resGo[1].Value, *resGo2[1].Value)
	zap.L().Info("", zap.Any("", *resGo[0].Delta))
	zap.L().Info("", zap.Any("", *resGo[0].Delta))
	zap.L().Info("", zap.Any("", *resGo2[1].Value))
	zap.L().Info("", zap.Any("", *resGo2[1].Value))
}
