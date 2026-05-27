pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
    skipDefaultCheckout(true)
    buildDiscarder(logRotator(numToKeepStr: '10'))
  }

  environment {
    APP_NAME = 'expense-manager-backend'
    BACKEND_DIR = '.'
    DOCKER_REGISTRY = 'docker.io'
    DOCKER_REPOSITORY = 'hungtd1192002/expanse_manager_be'
    DOCKER_CREDENTIALS_ID = 'docker-registry-credentials'
    SSH_CREDENTIALS_ID = 'aws-ec2-ssh-key'
    MYSQL_DSN_CREDENTIALS_ID = 'backend-mysql-dsn'
    DEPLOY_HOST = '13.212.161.182'
    CONTAINER_NAME = 'expense-manager-backend'
    DOCKER_NETWORK = 'expense_manager_be_default'
    HOST_PORT = '3000'
    CONTAINER_PORT = '3000'
    STORE_DRIVER = 'mysql'
    PUBLIC_DIR = '/app/public'
    IMAGE_TAG = ''
    DOCKER_IMAGE = ''
    DOCKER_IMAGE_LATEST = ''
  }

  stages {
    stage('Checkout source') {
      steps {
        checkout scm
      }
    }

    stage('Prepare') {
      steps {
        script {
          env.BACKEND_DIR = fileExists('backend/go.mod') ? 'backend' : '.'
          env.IMAGE_TAG = env.BUILD_NUMBER
          env.DOCKER_IMAGE = "${env.DOCKER_REGISTRY}/${env.DOCKER_REPOSITORY}:${env.IMAGE_TAG}"
          env.DOCKER_IMAGE_LATEST = "${env.DOCKER_REGISTRY}/${env.DOCKER_REPOSITORY}:latest"
        }
      }
    }

    stage('Test') {
      steps {
        dir(env.BACKEND_DIR) {
          sh '''
            docker run --rm \
              -v "$PWD":/src \
              -w /src \
              golang:1.26-alpine \
              sh -c "go test ./..."
          '''
        }
      }
    }

    stage('Build Docker image') {
      steps {
        dir(env.BACKEND_DIR) {
          sh '''
            docker build \
              -t "$DOCKER_IMAGE" \
              -t "$DOCKER_IMAGE_LATEST" \
              .
          '''
        }
      }
    }

    stage('Push Docker image') {
      steps {
        withCredentials([usernamePassword(
          credentialsId: env.DOCKER_CREDENTIALS_ID,
          usernameVariable: 'DOCKER_USERNAME',
          passwordVariable: 'DOCKER_PASSWORD'
        )]) {
          sh '''
            echo "$DOCKER_PASSWORD" | docker login "$DOCKER_REGISTRY" -u "$DOCKER_USERNAME" --password-stdin
            docker push "$DOCKER_IMAGE"
            docker push "$DOCKER_IMAGE_LATEST"
            docker logout "$DOCKER_REGISTRY"
          '''
        }
      }
    }

    stage('Deploy to AWS instance') {
      steps {
        withCredentials([
          usernamePassword(
            credentialsId: env.DOCKER_CREDENTIALS_ID,
            usernameVariable: 'DOCKER_USERNAME',
            passwordVariable: 'DOCKER_PASSWORD'
          ),
          sshUserPrivateKey(
            credentialsId: env.SSH_CREDENTIALS_ID,
            keyFileVariable: 'SSH_KEY',
            usernameVariable: 'SSH_USER'
          ),
          string(
            credentialsId: env.MYSQL_DSN_CREDENTIALS_ID,
            variable: 'MYSQL_DSN'
          )
        ]) {
          sh '''
            printf '%s' "$DOCKER_PASSWORD" | ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$SSH_USER@$DEPLOY_HOST" "docker login '$DOCKER_REGISTRY' -u '$DOCKER_USERNAME' --password-stdin"

            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$SSH_USER@$DEPLOY_HOST" "
              set -e
              docker network inspect '$DOCKER_NETWORK' >/dev/null 2>&1 || docker network create '$DOCKER_NETWORK'
              docker pull '$DOCKER_IMAGE'
              docker stop '$CONTAINER_NAME' || true
              docker rm '$CONTAINER_NAME' || true
              docker run -d \
                --name '$CONTAINER_NAME' \
                --restart unless-stopped \
                --network '$DOCKER_NETWORK' \
                -p '$HOST_PORT':'$CONTAINER_PORT' \
                -e PORT='$CONTAINER_PORT' \
                -e STORE_DRIVER='$STORE_DRIVER' \
                -e MYSQL_DSN='$MYSQL_DSN' \
                -e PUBLIC_DIR='$PUBLIC_DIR' \
                '$DOCKER_IMAGE'
              docker image prune -f
              docker logout '$DOCKER_REGISTRY'
            "
          '''
        }
      }
    }
  }

  post {
    success {
      echo "Backend build completed: ${APP_NAME}"
    }
    failure {
      echo "Backend build failed: ${APP_NAME}"
    }
  }
}
