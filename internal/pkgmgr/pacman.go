package pkgmgr

func newPacmanManager(lookPath lookPathFunc) Manager {
	return newCommandManager("pacman", lookPath)
}
