package flashcard

import "math"

// TargetAfterSession is a conservative application policy, not an FSRS parameter.
// The word scores are AoA proxies, so move slowly and require a meaningful sample.
func TargetAfterSession(target float64, answered, recalled int) float64 {
	if answered < 10 {
		return target
	}
	accuracy := float64(recalled) / float64(answered)
	if accuracy >= .85 {
		target += 2
	}
	if accuracy < .60 {
		target -= 2
	}
	return math.Max(0, math.Min(100, target))
}

func learnedSQL(alias string) string {
	return alias + ".state=2 AND " + alias + ".known_count>=3 AND " + alias + ".scheduled_days>=7"
}
