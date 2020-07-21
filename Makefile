install:
	@pipenv install ; \

test-all:
	./hack/run-tests.sh all

test-features:
	FEATURES="$(FEATURES)" ./hack/run-tests.sh features