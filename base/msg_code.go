package base

/*
提示消息号
*/
const (
	Success                = "成功"
	ParamError             = "参数错误"
	ServiceBusy            = "服务繁忙" //数据库访问达到上限
	SendGiftFailed         = "送礼失败"
	GiftTypeNotExist       = "礼物类型不存在"
	RankUpdateFail         = "排行榜更新失败"
	RankWriteToMongoFail   = "排行榜写进mongo失败"
	RankReadFail           = "获取排行榜数据失败"
	GiftLogReadFail        = "获取送礼记录失败"
	GiftLogLoadToRedisFail = "送礼记录加载到redis失败"
	RankLoadToRedisFail    = "排行榜加载到redis失败"
)
