pipeline {
  // `agent any` 表示该流水线可在任意可用节点执行。
  // 生产环境更推荐使用带标签的节点（例如 `agent { label 'docker' }`），
  // 以确保执行节点已经安装 Docker/kubectl 等工具。
  agent any

  options {
    // 为每条日志增加时间戳，便于排障和追踪耗时。
    timestamps()
    // 禁止同一个任务/分支并发构建，避免多个构建互相覆盖部署结果。
    // 当代码提交频繁时，这个选项非常有用。
    disableConcurrentBuilds()
    // 限制保留的历史构建和产物数量，控制 Jenkins 磁盘占用。
    buildDiscarder(logRotator(numToKeepStr: '30', artifactNumToKeepStr: '10'))
  }

  parameters {
    // 本次是否执行镜像构建与推送。
    // 设为 false 时，可作为“仅测试”流水线运行。
    booleanParam(name: 'PUSH_IMAGES', defaultValue: true, description: 'Build and push backend/frontend images to Docker Hub')
    // 是否执行 Kubernetes 部署（可选）。
    // 实际部署还会额外受 master 分支条件限制。
    booleanParam(name: 'DEPLOY_K8S', defaultValue: false, description: 'Deploy to Kubernetes after image push (master only)')
    // Docker 命名空间，通常是 Docker Hub 用户名或组织名。
    // 例如：your-name/social-app-backend:abcd1234
    string(name: 'DOCKER_NAMESPACE', defaultValue: 'your-dockerhub-username', description: 'Docker Hub namespace/username')
    // Kubernetes 命名空间（Deployment 等资源所在的 namespace）。
    string(name: 'K8S_NAMESPACE', defaultValue: 'social-app', description: 'Kubernetes namespace for deployment')
  }

  environment {
    // 在环境变量里统一定义镜像名，后续阶段复用，避免重复拼接。
    // 镜像 tag 会在构建阶段动态追加为 commit SHA。
    BACKEND_IMAGE = "${params.DOCKER_NAMESPACE}/social-app-backend"
    FRONTEND_IMAGE = "${params.DOCKER_NAMESPACE}/social-app-frontend"
  }

  stages {
    stage('Checkout') {
      steps {
        // 从 Jenkins 任务里配置的 SCM 拉取代码。
        // 对 Multibranch Pipeline 来说，会检出当前触发分支的最新提交。
        checkout scm
      }
    }

    stage('Test') {
      // 后端与前端测试并行执行，缩短总流水线时长。
      parallel {
        stage('Backend Test') {
          steps {
            dir('backend') {
              // 命令返回非 0 则该分支阶段失败（fail-fast）。
              sh 'go test ./...'
            }
          }
        }

        stage('Frontend Test') {
          steps {
            dir('frontend') {
              // 基于 package-lock.json 做可复现安装。
              sh 'npm ci'
              // 当前项目的 npm test 实际执行 vitest。
              sh 'npm test'
            }
          }
        }
      }
    }

    stage('Build & Push Images') {
      when {
        // 当 PUSH_IMAGES=false 时跳过本阶段。
        expression { return params.PUSH_IMAGES }
      }
      steps {
        script {
          // 使用不可变的 commit SHA 作为镜像 tag，便于追溯与回滚。
          // 示例：09b2c2a1d2e3
          env.IMAGE_TAG = sh(script: 'git rev-parse --short=12 HEAD', returnStdout: true).trim()
        }

        // Jenkins 凭证要求：
        // ID: dockerhub-creds
        // 类型：Username with password
        withCredentials([usernamePassword(credentialsId: 'dockerhub-creds', usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_TOKEN')]) {
          sh '''
            set -eux
            # 登录 Docker Hub，一次完成两套镜像构建与推送，最后退出登录。
            echo "$DOCKER_TOKEN" | docker login -u "$DOCKER_USER" --password-stdin

            # 构建并推送 backend 镜像（使用 SHA 不可变标签）。
            docker build -f backend/Dockerfile -t "$BACKEND_IMAGE:$IMAGE_TAG" .
            docker push "$BACKEND_IMAGE:$IMAGE_TAG"

            # 构建并推送 frontend 镜像（使用 SHA 不可变标签）。
            docker build -f frontend/Dockerfile -t "$FRONTEND_IMAGE:$IMAGE_TAG" .
            docker push "$FRONTEND_IMAGE:$IMAGE_TAG"

            # 仅 master 分支更新 latest 标签，
            # 避免功能分支意外覆盖稳定版本。
            if [ "$BRANCH_NAME" = "master" ]; then
              docker tag "$BACKEND_IMAGE:$IMAGE_TAG" "$BACKEND_IMAGE:latest"
              docker tag "$FRONTEND_IMAGE:$IMAGE_TAG" "$FRONTEND_IMAGE:latest"
              docker push "$BACKEND_IMAGE:latest"
              docker push "$FRONTEND_IMAGE:latest"
            fi

            docker logout
          '''
        }
      }
    }

    stage('Deploy to Kubernetes') {
      when {
        allOf {
          // 只允许 master 分支进入部署阶段。
          branch 'master'
          // 同时要求 PUSH_IMAGES=true 且 DEPLOY_K8S=true。
          expression { return params.PUSH_IMAGES && params.DEPLOY_K8S }
        }
      }
      steps {
        // Jenkins 凭证要求：
        // ID: kubeconfig-prod
        // 类型：Secret file（内容为 kubeconfig）
        withCredentials([file(credentialsId: 'kubeconfig-prod', variable: 'KUBECONFIG')]) {
          sh '''
            set -eux
            # 将 deployment 的容器镜像更新到本次构建的 SHA 标签。
            # deployment 名称和容器名需与你集群清单保持一致。
            kubectl -n "$K8S_NAMESPACE" set image deployment/social-backend backend="$BACKEND_IMAGE:$IMAGE_TAG"
            kubectl -n "$K8S_NAMESPACE" set image deployment/social-frontend frontend="$FRONTEND_IMAGE:$IMAGE_TAG"
            # 等待滚动发布完成，超时则流水线失败。
            kubectl -n "$K8S_NAMESPACE" rollout status deployment/social-backend --timeout=180s
            kubectl -n "$K8S_NAMESPACE" rollout status deployment/social-frontend --timeout=180s
          '''
        }
      }
    }
  }

  post {
    always {
      // 每次构建结束都清理工作目录，减少磁盘占用并避免脏文件影响下次构建。
      cleanWs()
    }
  }
}
