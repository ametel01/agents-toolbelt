package pkgmgr

func newAptManager(lookPath lookPathFunc) Manager {
	return newCommandManager("apt", lookPath)
}
