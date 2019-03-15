stage-all: clean stage-deploy
test-all: clean test-deploy
prod-all: clean prod-deploy

build:
	@echo '--- Building actions function ---'
	GOOS=linux go build lambda-actions/actions.go

zip_lambda: build
	@echo '--- Zip actions function ---'
	zip actions.zip ./actions

test-deploy: zip_lambda
	@echo '--- Build lambda test ---'
	@echo 'Package template'
	sam package --template-file actions-template.yaml --s3-bucket ringoid-cloudformation-template --output-template-file actions-template-packaged.yaml
	@echo 'Deploy test-actions-stack'
	sam deploy --template-file actions-template-packaged.yaml --s3-bucket ringoid-cloudformation-template --stack-name test-actions-stack --capabilities CAPABILITY_IAM --parameter-overrides Env=test --no-fail-on-empty-changeset

stage-deploy: zip_lambda
	@echo '--- Build lambda stage ---'
	@echo 'Package template'
	sam package --template-file actions-template.yaml --s3-bucket ringoid-cloudformation-template --output-template-file actions-template-packaged.yaml
	@echo 'Deploy stage-actions-stack'
	sam deploy --template-file actions-template-packaged.yaml --s3-bucket ringoid-cloudformation-template --stack-name stage-actions-stack --capabilities CAPABILITY_IAM --parameter-overrides Env=stage --no-fail-on-empty-changeset

prod-deploy: zip_lambda
	@echo '--- Build lambda prod ---'
	@echo 'Package template'
	sam package --template-file actions-template.yaml --s3-bucket ringoid-cloudformation-template --output-template-file actions-template-packaged.yaml
	@echo 'Deploy prod-actions-stack'
	sam deploy --template-file actions-template-packaged.yaml --s3-bucket ringoid-cloudformation-template --stack-name prod-actions-stack --capabilities CAPABILITY_IAM --parameter-overrides Env=prod --no-fail-on-empty-changeset

clean:
	@echo '--- Delete old artifacts ---'
	rm -rf actions.zip
	rm -rf actions

