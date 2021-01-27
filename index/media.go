package index

type Media []*Medium

func (media Media) Same(medium *Medium) *Medium {
	for _, m := range media {
		if m.Same(medium) {
			return m
		}
	}
	return nil
}
