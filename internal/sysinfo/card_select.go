package sysinfo

import "slices"

// PreferredDiscrete returns the card that should be preferred on a hybrid
// (laptop) graphics system, or nil if there is nothing to choose.
//
// The kernel numbers DRM cards in probe order, which is not stable across
// boots, so the choice must not depend on where a card is in the list.
// Neither can it rely on [Card.Embedded] alone: some NVIDIA drivers expose
// a connector named "eDP" on the discrete card, which makes every card
// look embedded. Instead, the discrete card is the one that is not the
// firmware's default (boot) GPU, which is also how switcheroo-control
// tells them apart.
func PreferredDiscrete(cards []Card) *Card {
	if !slices.ContainsFunc(cards, func(c Card) bool { return c.Embedded }) {
		return nil // not a laptop, nothing to offload from
	}

	if slices.ContainsFunc(cards, func(c Card) bool { return c.BootVGA }) {
		for i := range cards {
			if !cards[i].BootVGA {
				return &cards[i]
			}
		}
		return nil
	}

	// No boot GPU information is available, fall back to assuming that
	// the first card is the integrated one.
	if len(cards) >= 2 && cards[0].Embedded {
		return &cards[1]
	}
	return nil
}
