package domain

//go:generate go tool mockgen -destination mocks.go -typed -package domain . Hub,AdminHub,Bus
