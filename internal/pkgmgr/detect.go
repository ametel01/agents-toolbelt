package pkgmgr

import (
	"fmt"
	"os/exec"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

type lookPathFunc func(string) (string, error)

// managerPriority defines the supported package manager preference order.
var managerPriority = []string{"brew", "apt", "dnf", "pacman", "snap", "go", "pipx", "cargo"}

// DetectManagers discovers supported package managers available on PATH.
func DetectManagers() []Manager {
	return detectManagers(exec.LookPath)
}

func detectManagers(lookPath lookPathFunc) []Manager {
	candidates := []Manager{
		newBrewManager(lookPath),
		newAptManager(lookPath),
		newDNFManager(lookPath),
		newPacmanManager(lookPath),
		newSnapManager(lookPath),
		newGoManager(lookPath),
		newPipxManager(lookPath),
		newCargoManager(lookPath),
	}

	available := make([]Manager, 0, len(candidates))
	for _, manager := range candidates {
		if manager.Available() {
			available = append(available, manager)
		}
	}

	return available
}

// SelectBest picks the first install method that matches the available managers.
func SelectBest(tool catalog.Tool, available []Manager) (catalog.InstallMethod, Manager, error) {
	if len(available) == 0 {
		return catalog.InstallMethod{}, nil, ErrNoManagersDetected
	}

	methodsByManager := make(map[string]catalog.InstallMethod, len(tool.InstallMethods))
	for _, method := range tool.InstallMethods {
		methodsByManager[method.Manager] = method
	}

	for _, managerName := range managerPriority {
		for _, manager := range available {
			if manager.Name() != managerName {
				continue
			}

			method, ok := methodsByManager[manager.Name()]
			if ok {
				return method, manager, nil
			}
		}
	}

	return catalog.InstallMethod{}, nil, fmt.Errorf("%w for tool %s", ErrNoMatchingMethod, tool.ID)
}
