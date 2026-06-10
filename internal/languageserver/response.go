package languageserver

type userStatusResponse struct {
	UserStatus userStatus `json:"userStatus"`
}

type userStatus struct {
	Name                   string                 `json:"name"`
	Email                  string                 `json:"email"`
	UserTier               userTier               `json:"userTier"`
	PlanStatus             planStatus             `json:"planStatus"`
	CascadeModelConfigData cascadeModelConfigData `json:"cascadeModelConfigData"`
}

type userTier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type planStatus struct {
	AvailablePromptCredits flexInt64 `json:"availablePromptCredits"`
	AvailableFlowCredits   flexInt64 `json:"availableFlowCredits"`
	PlanInfo               planInfo  `json:"planInfo"`
}

type planInfo struct {
	MonthlyPromptCredits flexInt64 `json:"monthlyPromptCredits"`
	MonthlyFlowCredits   flexInt64 `json:"monthlyFlowCredits"`
}

type cascadeModelConfigData struct {
	ClientModelConfigs []clientModelConfig `json:"clientModelConfigs"`
}

type clientModelConfig struct {
	Label        string       `json:"label"`
	ModelOrAlias modelOrAlias `json:"modelOrAlias"`
	QuotaInfo    quotaInfo    `json:"quotaInfo"`
}

type modelOrAlias struct {
	Model string `json:"model"`
}

type quotaInfo struct {
	RemainingFraction *float64 `json:"remainingFraction"`
	ResetTime         string   `json:"resetTime"`
}
