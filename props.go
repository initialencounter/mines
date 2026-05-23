package main

type PropDefinition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"` // 0=active, 1=passive
}

const (
	PropDetector    = 101
	PropXJBD        = 102
	PropDoubleScore = 1001
	PropShield      = 1002

	PropTypeActive  = 0
	PropTypePassive = 1
)

var PropDefs = map[int]PropDefinition{
	PropDetector:    {PropDetector, "探测仪", PropTypeActive},
	PropXJBD:        {PropXJBD, "雷之奥义", PropTypeActive},
	PropDoubleScore: {PropDoubleScore, "双倍积分", PropTypePassive},
	PropShield:      {PropShield, "护盾", PropTypePassive},
}
