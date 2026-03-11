package pkgmgr

func newPipxManager(lookPath lookPathFunc) Manager {
	return newCommandManager("pipx", lookPath)
}
