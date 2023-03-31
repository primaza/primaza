PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))


.PHONY: all
all: help
	@:

include make/*.mk

ifneq ($(words $(MAKECMDGOALS)),0)
TARGET := $(word 1,$(MAKECMDGOALS))

ifeq ($(TARGET),primaza)
.PHONY: primaza
ifeq ($(words $(MAKECMDGOALS)),1)
primaza: help
else
primaza:
endif
	@:

include make/primaza/*.mk

else ifeq ($(TARGET),agentapp)
.PHONY: agentapp
ifeq ($(words $(MAKECMDGOALS)),1)
agentapp: help
else
agentapp:
endif
	@:

include make/agents/app/*.mk

else ifeq ($(TARGET),agentsvc)
.PHONY: agentsvc
ifeq ($(words $(MAKECMDGOALS)),1)
agentsvc: help
else
agentsvc:
endif
	@:

include make/agents/svc/*.mk

else ifeq ($(shell expr $(MAKE_VERSION) \>= 4.0), 1)
undefine TARGET
endif

endif



