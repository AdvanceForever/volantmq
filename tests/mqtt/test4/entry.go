package test4

import (
	"testing"

	"fmt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	assert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/troian/surgemq/tests/mqtt/config"
	testTypes "github.com/troian/surgemq/tests/types"
)

type impl struct {
}

var _ testTypes.Provider = (*impl)(nil)

const (
	testName = "persistence"
)

// nolint: golint
func New() testTypes.Provider {
	return &impl{}
}

// nolint: golint
func (im *impl) Name() string {
	return testName
}

// nolint: golint
func (im *impl) Run(t *testing.T) {
	worker := func(qos byte) {
		test_topic := "Persistence test 1"
		subsQos := byte(2)

		defaultPublishHandler := func(_ MQTT.Client, msg MQTT.Message) {
			assert.Condition(t, func() bool {
				if msg.Qos() == 2 && msg.Duplicate() {
					return false
				}
				return true
			}, "No duplicates should be received for qos 2")
		}

		cfg := config.Get()

		opts := MQTT.NewClientOptions().
			AddBroker(cfg.Host).
			SetClientID("xrctest1_test_4").
			SetCleanSession(true).
			SetUsername(cfg.TestUser).
			SetPassword(cfg.TestPassword).
			SetAutoReconnect(false).
			SetKeepAlive(20).
			SetDefaultPublishHandler(defaultPublishHandler).
			SetConnectionLostHandler(func(client MQTT.Client, reason error) {
				assert.Fail(t, reason.Error())
			})

		// Cleanup by connecting clean session
		c := MQTT.NewClient(opts)
		token := c.Connect()
		token.Wait()
		require.NoError(t, token.Error())
		c.Disconnect(250)

		opts.SetCleanSession(false)
		c = MQTT.NewClient(opts)
		token = c.Connect()
		token.Wait()
		require.NoError(t, token.Error())

		token = c.Subscribe(test_topic, subsQos, func(client MQTT.Client, msg MQTT.Message) {
			defaultPublishHandler(client, msg)
		})

		token.Wait()
		require.NoError(t, token.Error())

		for i := 0; i < 3; i++ {
			payload := fmt.Sprintf("Message sequence no %d", i)
			c.Publish(test_topic, qos, false, payload)
		}

		c.Disconnect(0)

		c = MQTT.NewClient(opts)
		token = c.Connect()
		token.Wait()
		require.NoError(t, token.Error())

		token = c.Unsubscribe(test_topic)
		token.Wait()
		require.NoError(t, token.Error())

		c.Disconnect(250)
	}

	//worker(1)
	worker(2)
}
