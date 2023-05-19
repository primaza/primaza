##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help
	@awk -v t=$(TARGET) 'BEGIN {if (length(t) != 0) s=" "; printf "\nUsage:\n make %s%s\033[36m<target>\033[0m\n", t, s }'
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


MDBOOK_VERSION ?= v0.4.28

.PHONY: book
book: ## builds The Primaza Book
	docker run --rm -v $(PROJECT_DIR)/docs/book:/book -w /book -u $$(id -u):$$(id -g) peaceiris/mdbook:$(MDBOOK_VERSION) build
