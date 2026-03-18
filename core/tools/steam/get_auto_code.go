package steam_tools

type SteamAuthInfo struct {
	DeviceID        string  `json:"device_id"`         // 设备 ID，格式可能已 URL 编码
	IdentitySecret  string  `json:"identity_secret"`   // 身份密钥（用于确认交易）
	SharedSecret    string  `json:"shared_secret"`     // 用于生成 TOTP 验证码的密钥
	PhoneNumberHint *string `json:"phone_number_hint"` // 手机号提示，可为空
	RevocationCode  string  `json:"revocation_code"`   // 用于撤销身份令牌的代码
	Secret1         string  `json:"secret_1"`          // 可能是另一个身份密钥，用于其他用途
	SerialNumber    string  `json:"serial_number"`     // 序列号
	//ServerTime      int64   `json:"server_time"`       // 服务器时间戳（秒）
	TokenGID string `json:"token_gid"` // Token 的 GID（可能用于身份识别）
	URI      string `json:"uri"`       // 用于 OTP 的 otpauth URI
}
