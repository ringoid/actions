all: clean stage-deploy

build:
	@echo '--- Building warmup-image function ---'
	GOOS=linux go build lambda-warmup/warm_up.go
	@echo '--- Building actions function ---'
	GOOS=linux go build lambda-actions/actions.go

zip_lambda: build
	@echo '--- Zip warmup-actions function ---'
	zip warmup-actions.zip ./warm_up
	@echo '--- Zip actions function ---'
	zip actions.zip ./actions

stage-deploy: zip_lambda
	@echo '--- Build lambda stage ---'
	@echo 'Package template'
	sam package --template-file actions-template.yaml --s3-bucket ringoid-cloudformation-template --output-template-file actions-template-packaged.yaml
	@echo 'Deploy stage-actions-stack'
	sam deploy --template-file actions-template-packaged.yaml --s3-bucket ringoid-cloudformation-template --stack-name stage-actions-stack --capabilities CAPABILITY_IAM --parameter-overrides Env=stage --no-fail-on-empty-changeset

clean:
	@echo '--- Delete old artifacts ---'
	rm -rf warmup-actions.zip
	rm -rf warm_up
	rm -rf actions.zip
	rm -rf actions

