package main

import (
	"time"
)

type GoadminOperationLog struct {
	Id        int64     `gorm:"column:id" json:"id"`
	UserId    int64     `gorm:"column:user_id" json:"user_id"`
	Path      string    `gorm:"column:path" json:"path"`
	Method    string    `gorm:"column:method" json:"method"`
	Ip        string    `gorm:"column:ip" json:"ip"`
	Input     string    `gorm:"column:input" json:"input"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"` //hahah
}

func (g *GoadminOperationLog) TableName() string {
	return "demo.goadmin_operation_log1"
}

func (n *Node) TableName() string {
	return "yebao_game_prod.node1"
}

type Node struct {
	Id              int       `gorm:"primarykey" json:"id"`
	NodeName        string    `gorm:"node_name" json:"node_name"`
	HostName        string    `gorm:"host_name" json:"host_name"`
	HostConfig      string    `gorm:"host_config" json:"host_config"`
	RouterInfo      string    `gorm:"router_info" json:"router_info"`
	Bandwidth       string    `gorm:"bandwidth" json:"bandwidth"`
	Region          string    `gorm:"region" json:"region"`
	NodeStatus      int       `gorm:"node_status" json:"node_status"` // 1开启2关闭
	NodeNum         int       `gorm:"node_num" json:"node_num"`
	NodeMaxNum      int       `gorm:"node_max_num" json:"node_max_num"`
	NodeIsMax       int       `gorm:"node_is_max" json:"node_is_max"` // 1：没有 2：已达到
	CreateDate      time.Time `gorm:"create_date" json:"create_date"`
	UpdateDate      time.Time `gorm:"update_date" json:"update_date"`
	CircuitId       int       `gorm:"circuit_id" json:"circuit_id"`
	OperatorId      int       `gorm:"operator_id" json:"operator_id"`
	NodeIp          string    `gorm:"node_ip" json:"node_ip"`
	NodeType        int       `gorm:"node_type" json:"node_type"`                 // 1, "国内" 2, "海外",3, "环网",4 nat海外，5pc回国
	Remark          string    `gorm:"remark" json:"remark"`                       // 备注
	ProvinceId      int       `gorm:"column:province_id" json:"province_id"`      // 省份
	OutIp           string    `gorm:"out_ip" json:"out_ip"`                       // 出口ip
	Tunnel          string    `gorm:"tunnel" json:"tunnel"`                       // 隧道配置
	ClientType      int       `gorm:"client_type" json:"client_type"`             // 0 PC 1手游
	Delay           int       `gorm:"delay" json:"delay"`                         // 增加延迟
	MobileAccelMode string    `gorm:"mobile_accel_mode" json:"mobile_accel_mode"` // 移动端，支持加速协议。 all / socks / open
	OutIpAddressId  int       `gorm:"out_ip_address_id" json:"out_ip_address_id"` // 出口ip的国家
}
