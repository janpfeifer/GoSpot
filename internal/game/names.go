package game

import "math/rand"

var tableNames = []string{
	"Milliways", "Wuthering Heights", "Arrakis", "Hogwarts", "Mordor",
	"Rivendell", "Shire", "Narnia", "Westeros", "Gondor",
	"Rohan", "Winterfell", "King's Landing", "Asgard", "Wakanda",
	"Atlantis", "El Dorado", "Camelot", "Avalon", "Neverland",
	"Wonderland", "Oz", "Krypton", "Gotham", "Metropolis",
	"Tatooine", "Endor", "Hoth", "Coruscant", "Naboo",
	"Pandora", "Cybertron", "Gallifrey", "Vulcan", "Romulus",
	"Qo'noS", "Deep Space Nine", "Babylon 5", "Caprica", "Kobol",
	"Trantor", "Terminus", "Solaris", "Dune", "Caladan",
	"Giedi Prime", "Kaitain", "Salusa Secundus", "Ix", "Tleilax",
	"Chapterhouse", "Osgiliath", "Minas Tirith", "Minas Morgul", "Isengard",
	"Helm's Deep", "Bree", "Lothlorien", "Mirkwood", "Erebor",
	"Dale", "Esgaroth", "Valinor", "Numenor", "Beleriand",
	"Doriath", "Gondolin", "Nargothrond", "Khazad-dum", "Moria",
	"Hogsmeade", "Diagon Alley", "Knockturn Alley", "Godric's Hollow", "Little Whinging",
	"Azkaban", "Durmstrang", "Beauxbatons", "Ilvermorny", "Macusa",
	"The Burrow", "Malfoy Manor", "Grimmauld Place", "Shell Cottage", "Spinners End",
	"The Daily Planet", "Wayne Manor", "Batcave", "Arkham Asylum", "Blackgate Penitentiary",
	"Themyscira", "Oa", "Apokolips", "New Genesis", "Rann",
	"Thanagar", "Daxam", "Colu", "Czarnia", "Tamaran",
}

// RandomTableName returns a random name from the list.
func RandomTableName() string {
	return tableNames[rand.Intn(len(tableNames))]
}
