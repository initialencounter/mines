package main

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"main/smtp"
	"os"
)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}
type MineConfig struct {
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
	Mines  int `mapstructure:"mines"`
}
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type Config struct {
	App      AppConfig       `mapstructure:"app"`
	Server   ServerConfig    `mapstructure:"server"`
	Mine     MineConfig      `mapstructure:"mine"`
	Database DatabaseConfig  `mapstructure:"database"`
	Smtp     smtp.MailConfig `mapstructure:"smtp"`
}

func getConfig() Config {
	// 设置配置文件名（不带扩展名）
	viper.SetConfigName("config")
	// 设置配置文件类型
	viper.SetConfigType("yaml")
	// 添加配置文件路径
	viper.AddConfigPath(".") // 表示当前目录
	configFile := "config.yml"
	if fileExists("config-test.yml") {
		configFile = "config-test.yml"
	}
	fmt.Println("Using config file:", configFile)
	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("Config file not found, creating default config.yml\n配置文件不存在，创建默认配置文件 config.yml")
		defaultConfig := []byte(`
app:
  name: "MineSweeper Server"
  version: "1.0.0"
server:
  port: 8080
  host: "localhost"
mine:
  width: 50
  height: 50
  mines: 600
database:
  host: "localhost"
  port: 3306
  user: "root"
  password: "mines"
smtp:
  host: "example.gmail.com"
  port: "587"
  tls: true
  username: ""
  password: ""
`)

		err := os.WriteFile(configFile, defaultConfig, 0644)
		if err != nil {
			log.Fatalf("Error creating config file, %s", err)
		}
		fmt.Println("Default config.yml created")
	}
	viper.SetConfigFile(configFile)
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var config Config

	// 将配置文件中的内容反序列化到结构体中
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return config
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
