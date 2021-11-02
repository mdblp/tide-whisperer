@Library('mdblp-library') _
def builderImage
pipeline {
    agent any
    stages {
        stage('Initialization') {
            steps {
                script {
                    utils.initPipeline()
                    if(env.GIT_COMMIT == null) {
                        // git commit id must be a 40 characters length string (lower case or digits)
                        env.GIT_COMMIT = "f".multiply(40)
                    }
                    env.RUN_ID = UUID.randomUUID().toString()
                    env.GOPRIVATE="github.com/mdblp/tide-whisperer-v2"
                }
            }
        }
        stage('Build') {
            agent {
                docker {
                    image 'docker.ci.diabeloop.eu/go-build:1.15'
                }
            }
            steps {
                withCredentials ([string(credentialsId: 'github-token', variable: 'GITHUB_TOKEN')]) {
                    sh 'git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"'
                    sh "$WORKSPACE/build.sh"
                    sh 'git config --global --unset url."https://${GITHUB_TOKEN}@github.com/".insteadOf'
                }
            }
        }
        stage('Test') {
            steps {
                echo 'start mongo to serve as a testing db'
                sh 'docker network create twtest${RUN_ID} && docker run --rm -d --net=twtest${RUN_ID} --name=mongo4twtest${RUN_ID} mongo:4.2'
                script {
                    docker.image('docker.ci.diabeloop.eu/go-build:1.15').inside("--net=twtest${RUN_ID}") {
                        sh "TIDEPOOL_STORE_ADDRESSES=mongo4twtest${RUN_ID}:27017 TIDEPOOL_STORE_DATABASE=data_test $WORKSPACE/test.sh"
                    }
                }
            }
            post {
                always {
                    sh 'docker stop mongo4twtest${RUN_ID} && docker network rm twtest${RUN_ID}'
                    junit 'test-report.xml'
                    archiveArtifacts artifacts: 'coverage.html', allowEmptyArchive: true
                }
            }
        }
        stage('Package') {
            steps {
                pack()
            }
        }
        stage('Documentation') {
            steps {
                genDocumentation()
            }
        }
        stage('Publish') {
            when { branch "dblp" }
            steps {
                publish()
            }
        }
    }
}
