package feature_flag

type FeatureFlag interface {
	Enabled(flagKey string) bool
}