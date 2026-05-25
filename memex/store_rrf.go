package memex

const rrfK = 60

// reciprocalRankFusion merges ranked ID lists using reciprocal rank fusion (k=60).
func reciprocalRankFusion(lists [][]string) []string {
	scores := make(map[string]float64)
	for _, list := range lists {
		if len(list) == 0 {
			continue
		}
		for rank, id := range list {
			scores[id] += 1.0 / float64(rrfK+rank+1)
		}
	}
	if len(scores) == 0 {
		return nil
	}
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sortIDsByScore(ids, scores)
	return ids
}

func sortIDsByScore(ids []string, scores map[string]float64) {
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 && scores[ids[j]] > scores[ids[j-1]] {
			ids[j], ids[j-1] = ids[j-1], ids[j]
			j--
		}
	}
}
