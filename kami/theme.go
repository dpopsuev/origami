package kami

// Theme provides domain-specific content for the Kami frontend.
// Consumers implement this interface to customize the debugger
// with their own agent intros, node descriptions, and visual assets.
type Theme interface {
	Name() string
	AgentIntros() []AgentIntro
	NodeDescriptions() map[string]string
	CostumeAssets() map[string]string
	CooperationDialogs() []Dialog
}

// AgentIntro is a single agent's introduction card content.
type AgentIntro struct {
	PersonaName string `json:"persona_name"`
	Element     string `json:"element"`
	Role        string `json:"role"`
	Catchphrase string `json:"catchphrase"`
}

// Dialog is a cooperation exchange between two agents.
type Dialog struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Message string `json:"message"`
}
