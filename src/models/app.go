package models

type AppInfo struct {
	Entity AppEntity `json:"entity"`
}

type AppEntity struct {
	Instances int `json:"instances"`
}
