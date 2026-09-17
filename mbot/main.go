package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"mbot/util"
	"os/exec"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttConfig struct {
	Broker   string
	ClientID string
	Username string
	Password string
	SubTopic string
	PubTopic string
}

var cfg = &MqttConfig{
	// Broker:   "broker.emqx.io:1883",
	Broker:   "127.0.0.1:1883",
	ClientID: "go-temperature-client",
	Username: "root",
	Password: "password",
	SubTopic: "sugar/home/cmd",
	PubTopic: "sugar/home/sensor/temperature",
}

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", cfg.Broker))
	opts.SetClientID(cfg.ClientID)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password) // 可选
	}
	opts.SetAutoReconnect(true)

	// 连接回调
	opts.OnConnect = func(c mqtt.Client) {
		slog.Info("Connected to MQTT broker", "broker", cfg.Broker)
		// 连接成功后自动订阅
		if token := c.Subscribe(cfg.SubTopic, 1, messageHandler); token.Wait() && token.Error() != nil {
			slog.Error("Subscribe error: %v", "err", token.Error())
		}
	}

	// 断开回调
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		slog.Error("Connection lost", "broker", cfg.Broker, "error", err)
	}

	// 创建客户端
	client := mqtt.NewClient(opts)

	// 连接
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Failed to connect to MQTT broker", "broker", cfg.Broker, "error", token.Error())
	}
	instanceInfo, err := util.CollectInstanceInfo()
	if err != nil {
		slog.Error("Failed to collect instance info", "error", err)
	}
	fmt.Println(util.Marshal(instanceInfo))

	// 模拟温度数据上传
	for {
		temp := getTemperature()

		// 构造消息（JSON）
		payload := fmt.Sprintf(`{"device_id":"sensor_001","temperature":%.2f,"ts":%d}`,
			temp, time.Now().Unix())

		// 发布消息
		token := client.Publish(cfg.PubTopic, 0, false, payload)
		token.Wait()
		slog.Info("Published:", "payload", payload)
		time.Sleep(30 * time.Second)
	}
}

// 模拟获取温度
func getTemperature() float64 {
	return 20 + (5 * float64(time.Now().Unix()%10) / 10)
}

type Payload struct {
	Msg string `json:"msg"`
	Cmd string `json:"cmd"`
}

type PayloadV2 struct {
	Result string `json:"result"`
}

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	slog.Info("Received message", "topic", msg.Topic())

	var data Payload
	err := json.Unmarshal(msg.Payload(), &data)
	if err != nil {
		slog.Error("Failed to parse message payload", "error", err)
		return
	}

	fmt.Printf("Received message - Msg: %s, Cmd: %s", data.Msg, data.Cmd)
	switch data.Msg {
	case "bash":
		var dataV2 PayloadV2
		stdout, stderr, _ := RunCommandWithTimeout(10, data.Cmd)
		if stdout != "" {
			dataV2.Result = stdout
		}
		if stderr != "" {
			dataV2.Result = stderr
		}
		jsonBytes, _ := json.Marshal(dataV2)
		token := client.Publish(cfg.PubTopic, 0, false, jsonBytes)
		token.Wait()

	default:
		fmt.Printf("payload: %v", string(msg.Payload()))
	}
}

func RunCommandWithTimeout(timeout int, command string) (stdout, stderr string, isKilled bool) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command("/bin/bash", "-c", command)

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Start()
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	after := time.After(time.Duration(timeout) * time.Second)
	select {
	case <-after:
		cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(10 * time.Millisecond)
		cmd.Process.Kill()
		isKilled = true
	case <-done:
		isKilled = false
	}
	stdout = string(bytes.TrimSpace(stdoutBuf.Bytes())) // Remove \n
	stderr = string(bytes.TrimSpace(stderrBuf.Bytes())) // Remove \n
	return
}
