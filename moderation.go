//go:build experiment
package openai

/*
import (
	"macbot/guidelines"
	"strings"
)

// moderationSysMessage is a "system" message that instructs AI to act as a moderator.
var moderationSysMessage string

var ruleReplacer = strings.NewReplacer(
	"1.", "1. ",
)

func init() {
	moderationSysMessage = `You are assisting in moderation of a Discord server called "My Anime Chat" (short "MAC").
Messages sent to you are user content pending moderation. Some may include a comment from a moderator prefixed by "Comment:".
Reply to every message with two separate paragraphs, first starting with "Analysis:" containing a brief review of the content, second starting with "Action:" containing a short advise to moderators on what to do regarding this content.
Possible actions include, but are not limited to:
* verbal warning - explaining member what they did wrong and asking them to mind it in future;
* mute - isolating a member in a separate channel to privately discuss the matter at hand and decide on further actions (unresponsive members are banned automatically);
* note - creating a note for moderators attached to the member to review later in case of further problems;
* warning - increasing a warning counter of a member, leading to possible automatic actions (3 warnings - temporary ban for 24 hours, 4 warnings - permanent ban, warnings may be removed later if member behaves well);
* kick - removing member from the server but leaving them the ability to join back;
* ban - removing member form server permanently.`

	rules := guidelines.FullText

	moderationSysMessage += "\n\n" + rules
}
*/

// based on responses and mentions
/*
{
	User2: Do you like Spy X Family?
	RE:User1: Nah.
	User3: Wasn't best I think.
	RE:User2: @Astrea what you think?
}
{
	Astrea: User2, I thonk it's a great show!
}
{
	RE:User2: Astrea, what's good about it?
}
*/
