//go:build experiment
package embedding

import (
	"macbot/framework"
	"testing"
)

func init() {
	framework.LoadConfig("../../config.json")
}

func TestOne(t *testing.T) {
	s1 := "This is a test string 1"
	s2 := "This is a test string 2"

	vs, err := Array(s1, s2)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
		return
	}

	t.Logf("Vector length: %d", len(vs[0]))

	diff := vs[0].AngleDiff(vs[1])
	t.Logf("Angle difference: %f", diff)
	dist := vs[0].Distance(vs[1])
	t.Logf("Distance: %f", dist)

	t.FailNow() // to show the output
}

var animeDescriptions = []string{
	`Darker than black - Action, Mystery, Sci-Fi
In Tokyo, an impenetrable field known as "Hell's Gate" appeared ten years ago. At the same time, psychics who wield paranormal powers at the cost of their conscience also emerged. Hei is one of the most powerful of these psychic agents, and along with his blind associate, Yin, works for one of the many rival agencies vying to unlock the mysteries of Hell's Gate.`,
	`Made in abyss - Adventure, Drama, Fantasy, Horror, Mystery, Sci-Fi
The "Abyss" is the last unexplored place in the world. Strange and wonderful creatures roam within, and it is full of precious relics that present humans can't recreate. Those that dare to explore the depths are known as "Cave Raiders." An orphan girl named Riko lives on the rim. Her dream is to become a Cave Raider like her mother and solve the mysteries of the cave system. One day, Riko starts exploring the caves and discovers a robot who resembles a human boy.`,
	`Mahou Shoujo Madoka☆Magica - Action, Drama, Fantasy, Mahou Shoujo, Psychological, Thriller
One night, 14-year-old Madoka Kaname has a terrible nightmare - against the backdrop of a devastated city, she witnesses a girl fight a losing battle against a dreadful being lingering above, while a cat-like magical creature tells Madoka the only way to change such tragic outcome is for her to make a contract with him and become a magical girl.
The next day, the teen's dream seemingly becomes reality as the girl she saw in her dream - Homura - arrives at Mitakihara Middle School as a transfer student, mysteriously warning Madoka to stay just the way she is; but when later on she and her best friend Sayaka encounter the same cat-like magical creature from her dream - who introduces himself as Kyubey - the pair discovers that magical girls are real, and what's more, they can choose to become one. All they must do is sign a contract with Kyubey and agree to take on the duty to fight abstract beings called 'witches' that spread despair to the human world, and in return, each one of them will be granted any single wish they desire. However, as Homura's omen suggests, there might be far more to becoming a magical girl than Madoka and Sayaka realize... `,
	`Kanata no Astra - Adventure, Mystery, Sci-Fi
Itʼs the first day of Planet Camp, and Aries Spring couldnʼt be more excited! She, along with eight other strangers, leave for Planet McPa for a weeklong excursion. Soon after they arrive, however, a mysterious orb appears and transports them into the depths of space, where they find an empty floating spaceship… `,
	`Dr. STONE - Action, Adventure, Comedy, Sci-Fi
After five years of harboring unspoken feelings, high-schooler Taiju Ooki is finally ready to confess his love to Yuzuriha Ogawa. Just when Taiju begins his confession however, a blinding green light strikes the Earth and petrifies mankind around the world— turning every single human into stone.
Several millennia later, Taiju awakens to find the modern world completely nonexistent, as nature has flourished in the years humanity stood still. Among a stone world of statues, Taiju encounters one other living human: his science-loving friend Senkuu, who has been active for a few months. Taiju learns that Senkuu has developed a grand scheme—to launch the complete revival of civilization with science. Taiju's brawn and Senkuu's brains combine to forge a formidable partnership, and they soon uncover a method to revive those petrified.
However, Senkuu's master plan is threatened when his ideologies are challenged by those who awaken. All the while, the reason for mankind's petrification remains unknown. `,
	`Horimiya - Comedy, Romance, Slice of Life
A secret life is the one thing they have in common. At school, Hori is a prim and perfect social butterfly, but the truth is she's a brash homebody. Meanwhile, under a gloomy facade, Miyamura hides a gentle heart, along with piercings and tattoos. In a chance meeting, they both reveal a side they've never shown. Could this blossom into something new?`,
	`Sono Bisque Doll wa Koi wo Suru - Comedy, Ecchi, Romance, Slice of Life
High schooler Wakana Gojou cares about one thing: making Hina dolls. With nobody to share his obsession, he has trouble finding friends—or even holding conversation. But after the school’s most popular girl, Marin Kitagawa, reveals a secret of her own, he discovers a new purpose for his sewing skills. Together, they’ll make her cosplay dreams come true! `,
	`Dungeon ni Deai wo Motomeru no wa Machigatteiru Darou ka - Action, Adventure, Comedy, Fantasy, Romance
Some adventurers delve into the sprawling labyrinths beneath the city of Orario to find fame and fortune. Others come to test their skills against the legions of monsters lurking in the darkness below. However, Bell Cranel’s grandfather told him a different reason: it’s a great place to rescue (and subsequently meet) girls! Now that Bell’s a dungeon delver himself, the ladies he’s encountering aren’t the helpless damsels in distress he’d imagined, and one of them, the beautiful swordswoman Ais Wallenstein, keeps rescuing Bell instead. As embarrassing as that is, it’s nothing compared to what happens when goddesses get involved. Freya, Hephaistos, and Loki, with their powerful Familias, are intimidating enough, but there’s one goddess whose relationship with Bell is certain to spark trouble.`,
}

func TestAnime(t *testing.T) {
	vs, err := Array(animeDescriptions...)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
		return
	}

	// show distance between each pair
	bestDiffScore := -1.0
	bestDiffPair := [2]string{}
	bestDistScore := 1000000.0
	bestDistPair := [2]string{}
	for i, v1 := range vs {
		for j, v2 := range vs {
			if i >= j {
				continue
			}
			diff := v1.AngleDiff(v2)
			dist := v1.Distance(v2)
			t.Logf("%f %f\n\t%s\n\t%s", dist, diff, animeDescriptions[i], animeDescriptions[j])
			if diff > bestDiffScore {
				bestDiffScore = diff
				bestDiffPair = [2]string{animeDescriptions[i], animeDescriptions[j]}
			}
			if dist < bestDistScore {
				bestDistScore = dist
				bestDistPair = [2]string{animeDescriptions[i], animeDescriptions[j]}
			}
		}
	}

	t.Logf("Best angle difference: %f\n\t%s\n\t%s", bestDiffScore, bestDiffPair[0], bestDiffPair[1])
	t.Logf("Best distance: %f\n\t%s\n\t%s", bestDistScore, bestDistPair[0], bestDistPair[1])

	t.FailNow() // to show the output
}
