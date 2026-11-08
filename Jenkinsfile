pipeline {
  agent any
  options { skipDefaultCheckout(); timestamps(); disableConcurrentBuilds(); timeout(time: 45, unit: 'MINUTES') }
  triggers { pollSCM('H/5 * * * *') }
  environment { SERVICE = 'ms-profiles'; IMAGE = 'rhm/ms-profiles' }
  stages {
    stage('Checkout') { steps { checkout scm; script { env.IMAGE_TAG = "${env.BUILD_NUMBER}-${sh(script: 'git rev-parse --short=7 HEAD', returnStdout: true).trim()}" } } }
    stage('Build') {
      agent { docker { image 'golang:1.26-alpine'; reuseNode true } }
      steps { sh 'go mod download'; sh 'go vet ./...'; sh 'go build ./cmd/...' }
    }
    stage('Test') {
      agent { docker { image 'golang:1.26-alpine'; reuseNode true } }
      steps { sh 'go mod download'; sh 'make test-ci' }
    }
    stage('Secret scan') { steps { sh 'docker run --rm -v "$PWD:/src" zricethezav/gitleaks:v8.24.0 detect --source /src --no-git --redact' } }
    stage('SonarQube') {
      agent { docker { image 'sonarsource/sonar-scanner-cli:11'; reuseNode true; args "--network ${env.RHM_CI_NETWORK}" } }
      steps { withSonarQubeEnv('sonarqube') { sh 'sonar-scanner -Dsonar.projectKey=rhm-ms-profiles -Dsonar.sources=cmd,internal' } }
    }
    stage('Quality Gate') { steps { timeout(time: 5, unit: 'MINUTES') { waitForQualityGate abortPipeline: true } } }
    stage('Package') { steps { sh 'docker build -t "$IMAGE:$IMAGE_TAG" -t "$IMAGE:latest" .' } }
    stage('Publish') { steps { sh 'docker tag "$IMAGE:$IMAGE_TAG" "$RHM_REGISTRY/$IMAGE:$IMAGE_TAG"; docker tag "$IMAGE:latest" "$RHM_REGISTRY/$IMAGE:latest"; docker push "$RHM_REGISTRY/$IMAGE:$IMAGE_TAG"; docker push "$RHM_REGISTRY/$IMAGE:latest"' } }
  }
}
