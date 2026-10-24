#!/usr/bin/env bash
# Nexogent — full spin-up.
# Builds and pushes the agent image, ensures the read-only IAM policy + IRSA
# service account, scales the node group up, and helm-installs Nexogent.
#
# READ-ONLY product: the agent only reads the account and reports.
# No static AWS keys are used in-cluster — access is via IRSA.
#
# Requires local AWS credentials (aws configure / assumed role), docker,
# kubectl, helm, eksctl. Never paste AWS keys into this script.
set -euo pipefail

# ---- config (override via env) ----
REGION="${REGION:-us-east-1}"
CLUSTER="${CLUSTER:-eks-lab}"
NODEGROUP="${NODEGROUP:-core}"
NAMESPACE="${NAMESPACE:-nexogent}"
REPO="${REPO:-nexogent}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
POLICY_NAME="${POLICY_NAME:-NexogentReadOnly}"
SA_NAME="${SA_NAME:-nexogent-sa}"
ROLE_NAME="${ROLE_NAME:-NexogentReadOnly}"
DESIRED_NODES="${DESIRED_NODES:-2}"

cd "$(dirname "$0")"

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ECR="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
IMAGE="${ECR}/${REPO}"
echo ">> Account ${ACCOUNT_ID} | region ${REGION} | cluster ${CLUSTER}"

# ---- 1. ECR repo ----
echo ">> Ensuring ECR repo ${REPO}"
aws ecr describe-repositories --repository-names "${REPO}" --region "${REGION}" >/dev/null 2>&1 \
  || aws ecr create-repository --repository-name "${REPO}" --region "${REGION}" >/dev/null
aws ecr get-login-password --region "${REGION}" \
  | docker login --username AWS --password-stdin "${ECR}"

# ---- 2. Build + push image ----
echo ">> Building and pushing ${IMAGE}:${IMAGE_TAG}"
docker build --platform linux/amd64 -t "${IMAGE}:${IMAGE_TAG}" .
docker push "${IMAGE}:${IMAGE_TAG}"

# ---- 3. Read-only IAM policy ----
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"
if aws iam get-policy --policy-arn "${POLICY_ARN}" >/dev/null 2>&1; then
  echo ">> IAM policy ${POLICY_NAME} exists"
else
  echo ">> Creating IAM policy ${POLICY_NAME}"
  aws iam create-policy --policy-name "${POLICY_NAME}" \
    --policy-document file://deploy/iam/nexogent-readonly-policy.json >/dev/null
fi

# ---- 4. OIDC + IRSA service account (read-only role) ----
echo ">> Associating OIDC provider (idempotent)"
eksctl utils associate-iam-oidc-provider --cluster "${CLUSTER}" --region "${REGION}" --approve

echo ">> Ensuring IRSA service account ${SA_NAME}"
eksctl create iamserviceaccount \
  --cluster "${CLUSTER}" --region "${REGION}" \
  --namespace "${NAMESPACE}" --name "${SA_NAME}" \
  --role-name "${ROLE_NAME}" \
  --attach-policy-arn "${POLICY_ARN}" \
  --approve --override-existing-serviceaccounts

ROLE_ARN="$(aws iam get-role --role-name "${ROLE_NAME}" --query 'Role.Arn' --output text)"
echo ">> IRSA role: ${ROLE_ARN}"

# ---- 5. Scale node group up ----
echo ">> Scaling node group ${NODEGROUP} to desired ${DESIRED_NODES}"
aws eks update-nodegroup-config --cluster-name "${CLUSTER}" --nodegroup-name "${NODEGROUP}" \
  --region "${REGION}" \
  --scaling-config "minSize=1,maxSize=4,desiredSize=${DESIRED_NODES}" >/dev/null 2>&1 || true

# ---- 6. Helm install ----
echo ">> helm upgrade --install nexogent"
helm upgrade --install nexogent deploy/helm/nexogent \
  --namespace "${NAMESPACE}" --create-namespace \
  --set image.repository="${IMAGE}" \
  --set image.tag="${IMAGE_TAG}" \
  --set serviceAccount.roleArn="${ROLE_ARN}" \
  --set region="${REGION}" \
  --wait --timeout 5m

echo ">> Done. Pods:"
kubectl -n "${NAMESPACE}" get pods,svc
echo
echo ">> Port-forward to test:"
echo "   kubectl -n ${NAMESPACE} port-forward svc/nexogent 8080:8080"
echo "   curl -s localhost:8080/health && curl -s -X POST localhost:8080/api/v1/review | head"
