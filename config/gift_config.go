package config

/*
礼物价值
*/
var GiftConfigMap map[int]int

func init() {
	GiftConfigMap = map[int]int{
		1: 10,
		2: 20,
		3: 30,
		4: 40,
		5: 50,
	}
}
