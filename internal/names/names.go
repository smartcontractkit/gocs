// Package names generates random human-readable names for changeset markdown files.
package names

import (
	"fmt"
	"math/rand/v2"
)

// Adjectives for changeset names.
var adjectives = []string{
	"brave", "calm", "cool", "dark", "deep",
	"dry", "fair", "fast", "flat", "free",
	"fresh", "full", "giant", "glad", "gold",
	"good", "gray", "green", "happy", "heavy",
	"hot", "huge", "kind", "large", "late",
	"lean", "light", "little", "long", "loud",
	"lucky", "mean", "mild", "neat", "new",
	"nice", "odd", "old", "orange", "plain",
	"proud", "quick", "quiet", "rare", "red",
	"rich", "ripe", "rough", "round", "sad",
	"safe", "sharp", "short", "shy", "silver",
	"simple", "slim", "slow", "small", "smart",
	"smooth", "soft", "sour", "spicy", "stale",
	"steep", "still", "strong", "sweet", "tall",
	"thin", "tight", "tiny", "tough", "warm",
	"weak", "wet", "wide", "wild", "wise",
	"young", "brave", "bright", "busy", "chilly",
	"clean", "clever", "cold", "crazy", "crisp",
	"cuddly", "curly", "damp", "dirty", "dusty",
	"early", "empty", "fancy", "fierce", "filthy",
	"fluffy", "fuzzy", "gentle", "grumpy", "healthy",
}

// Nouns for changeset names.
var nouns = []string{
	"ants", "bats", "bears", "bees", "birds",
	"cats", "cows", "crabs", "deer", "dogs",
	"doves", "ducks", "eagles", "eels", "elks",
	"fish", "foxes", "frogs", "geese", "goats",
	"hawks", "hens", "hogs", "horses", "jays",
	"lambs", "larks", "lions", "mice", "moles",
	"moths", "mules", "newts", "otters", "owls",
	"pandas", "pigs", "quails", "rams", "rats",
	"ravens", "seals", "sharks", "sheep", "skunks",
	"slugs", "snails", "snakes", "sparks", "spiders",
	"squid", "storks", "swans", "tigers", "toads",
	"trout", "tuna", "turkeys", "turtles", "vipers",
	"wasps", "waves", "whales", "wolves", "worms",
	"yaks", "zebras", "apples", "beans", "berries",
	"books", "boots", "boxes", "breads", "brooms",
	"cakes", "candles", "cards", "chairs", "clams",
	"clocks", "clouds", "coins", "combs", "cups",
	"doors", "drums", "eggs", "ferns", "fires",
	"flames", "flutes", "gems", "grapes", "hills",
	"horns", "kings", "kites", "knots", "lakes",
}

// Verbs for changeset names.
var verbs = []string{
	"act", "add", "aid", "aim", "ask",
	"bake", "bark", "beam", "bite", "blow",
	"boil", "bounce", "bow", "bump", "burn",
	"buzz", "call", "camp", "care", "carry",
	"catch", "chase", "cheer", "chew", "clap",
	"clean", "climb", "close", "cook", "copy",
	"count", "cover", "crash", "crawl", "cry",
	"curl", "dance", "dash", "dig", "dip",
	"dive", "drag", "draw", "dream", "dress",
	"drink", "drive", "drop", "drum", "dry",
	"dust", "earn", "eat", "end", "enter",
	"erase", "escape", "fade", "fail", "fall",
	"fear", "feel", "fetch", "fill", "find",
	"fish", "fit", "fix", "flap", "flash",
	"float", "flow", "fly", "fold", "follow",
	"form", "free", "freeze", "frown", "gain",
	"gallop", "gaze", "give", "glow", "grab",
	"greet", "grin", "grip", "grow", "guard",
	"guess", "guide", "hang", "heal", "hear",
	"help", "hide", "hike", "hit", "hold",
}

// Generate creates a random changeset name in the format "adjective-noun-verb".
// Using math/rand is acceptable here as we don't need cryptographic randomness for names.
func Generate() string {
	adj := adjectives[rand.IntN(len(adjectives))] //nolint:gosec // G404: non-security use case
	noun := nouns[rand.IntN(len(nouns))]          //nolint:gosec // G404: non-security use case
	verb := verbs[rand.IntN(len(verbs))]          //nolint:gosec // G404: non-security use case

	return fmt.Sprintf("%s-%s-%s", adj, noun, verb)
}
