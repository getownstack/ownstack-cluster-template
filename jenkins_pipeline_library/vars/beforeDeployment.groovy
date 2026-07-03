def call(String applicationId) {
    def values = values()
    def sha = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
    def branch = env.BRANCH_NAME ?: sh(script: 'git rev-parse --abbrev-ref HEAD', returnStdout: true).trim()
    branch = branch.replaceAll('[^a-zA-Z0-9_.-]', '-')
    def tag = "${branch}-${sha}"
    def imageNameWithTag = "${values.imageRepositoryProject}/${applicationId}:${tag}"
    def image = "${env.IMAGE_REGISTRY}/${imageNameWithTag}"
    def scriptPath = "infrastructure/${applicationId}/before-deployment.sh"

    container('helm') {
        if (!fileExists(scriptPath)) {
            echo "No ${scriptPath}; skipping before-deployment hook"
            return
        }

        echo "Running ${scriptPath}"
        sh """
            chmod +x '${scriptPath}'
            APPLICATION_ID='${applicationId}' \
            RELEASE_NAME='${applicationId}' \
            NAMESPACE='prod' \
            IMAGE='${image}' \
            IMAGE_REGISTRY='${env.IMAGE_REGISTRY}' \
            IMAGE_REPOSITORY_PROJECT='${values.imageRepositoryProject}' \
            IMAGE_TAG='${tag}' \
            '${scriptPath}'
        """
    }
}
