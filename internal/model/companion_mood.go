package model

// CompanionMood 是一个填写者的三组心情选项，固定选项使用稳定标识。
type CompanionMood struct {
	Statuses      []string `json:"statuses"`
	Reasons       []string `json:"reasons"`
	ReasonCustom  string   `json:"reason_custom"`
	Comforts      []string `json:"comforts"`
	ComfortCustom string   `json:"comfort_custom"`
}

// CompanionMoodView 附带所属关系，保存时用该标识检查绑定是否已更换。
type CompanionMoodView struct {
	BindingID int64 `json:"binding_id"`
	CompanionMood
}
