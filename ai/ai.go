package ai

import (
	"github.com/andyleap/gioframework"
)

type context struct {
	generalLocation int
}

func findGeneral(g *gioframework.Game) int {
	for i, c := range g.GameMap {
		if c.Faction == g.PlayerIndex &&
			c.Type == gioframework.General {
			return i
		}
	}
	return -1
}

func InitContext(g *gioframework.Game) context {
	// Find the general
	return context{
		generalLocation: findGeneral(g),
	}
}

func (*context) ProcessGame(g *gioframework.Game) {

}
