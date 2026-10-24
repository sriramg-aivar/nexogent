#!/usr/bin/env bash
# Nexogent — full spin-up (microservice architecture).
#
# Deploys THREE workloads into the client's EKS:
#   - nexogent-backend   (Go agent/API, read-only, IRSA)
#   - nexogent-frontend  (React dashboard served by Nginx)
#   - nexogent-prowler   (open-source Prowler scanner, CronJob, read-only, IRSA)
# plus a shared PVC (Prowler OCSF results) and an ALB Ingress (public URL).
#
# READ-ONLY product: it only reads the account and reports. No static AWS keys
# in-cluster — both backend and prowler use the read-only IRSA role.
#
# Requires local AWS credentials (aws configure / assumed role), docker,
# kubectl, helm, eksctl. Never paste AWS keys into this script.
set -euo pipefail

# ---- config (override via env) ----
REGION="${REGION:-us-east-1}"
CLUSTER="${CLUSTER:-eks-lab}"
NODEGROUP="${NODEGROUP:-core}"
NAMESPACE="${NAMESPACE:-nexogent}"
BACKEND_REPO="${BACKEND_REPO:-nexogent}"
FRONTEND_REPO="${FRONTEND_REPO:-nexogent-frontend}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
POLICY_NAME="${POLICY_NAME:-NexogentReadOnly}"
SA_NAME="${SA_NAME:-nexogent-sa}"
ROLE_NAME="${ROLE_NAME:-NexogentReadOnly}"
DESIRED_NODES="${DESIRED_NODES:-2}"

cd "$(dirname "$0")"

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ECR="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
BACKEND_IMAGE="${ECR}/${BACKEND_REPO}"
FRONTEND_IMAGE="${ECR}/${FRONTEND_REPO}"
echo ">> Account ${ACCOUNT_ID} | region ${REGION} | cluster ${CLUSTER}"

ecr_login() { aws ecr get-login-password --region "${REGION}" | docker login --username AWS --password-stdin "${ECR}"; }
push_retry() { # $1 = image:tag
  for i in 1 2 3 4 5; do
    if docker push "$1" 2>&1 | tee /dev/stderr | grep -q "digest:"; then return 0; fi
    echo ">> push retry $i for $1"; ecr_login >/dev/null 2>&1; sleep 3
  done
  echo "!! push failed: $1"; return 1
}

# ---- 1. ECR repos (one per image, private) ----
echo ">> Ensuring private ECR repos"
for r in "${BACKEND_REPO}" "${FRONTEND_REPO}"; do
  aws ecr describe-repositories --repository-names "$r" --region "${REGION}" >/dev/null 2>&1 \
    || aws ecr create-repository --repository-name "$r" --region "${REGION}" >/dev/null
done
ecr_login

# ---- 2. Build + push both images (amd64) ----
echo ">> Building + pushing backend ${BACKEND_IMAGE}:${IMAGE_TAG}"
docker build --platform linux/amd64 -t "${BACKEND_IMAGE}:${IMAGE_TAG}" .
push_retry "${BACKEND_IMAGE}:${IMAGE_TAG}"

echo ">> Building + pushing frontend ${FRONTEND_IMAGE}:${IMAGE_TAG}"
docker build --platform linux/amd64 -t "${FRONTEND_IMAGE}:${IMAGE_TAG}" ./dashboard
push_retry "${FRONTEND_IMAGE}:${IMAGE_TAG}"

# ---- 3. Read-only IAM policy ----
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"
if aws iam get-policy --policy-arn "${POLICY_ARN}" >/dev/null 2>&1; then
  echo ">> IAM policy ${POLICY_NAME} exists"
else
  echo ">> Creating IAM policy ${POLICY_NAME}"
  aws iam create-policy --policy-name "${POLICY_NAME}" \
    --policy-document file://deploy/iam/nexogent-readonly-policy.json >/dev/null
fi

# ---- 4. OIDC + IRSA service account (read-only role for backend + prowler) ----
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

# ---- 5. Cluster prerequisites: EBS CSI driver (PVC) + ALB controller (Ingress) ----
# These are required for the shared Prowler PVC and the public ALB URL.
# See deploy/CLUSTER_PREREQS.md to install them once per cluster.
if ! kubectl get csidriver ebs.csi.aws.com >/dev/null 2>&1; then
  echo "!! EBS CSI driver not found — the Prowler PVC won't bind."
  echo "   Install it (see deploy/CLUSTER_PREREQS.md) or run with --set prowler.enabled=false."
fi
if ! kubectl get ingressclass alb >/dev/null 2>&1; then
  echo "!! AWS Load Balancer Controller (ingressclass 'alb') not found — no public URL."
  echo "   Install it (see deploy/CLUSTER_PREREQS.md) or run with --set ingress.enabled=false."
fi

# ---- 6. Scale node group up ----
echo ">> Scaling node group ${NODEGROUP} to desired ${DESIRED_NODES}"
aws eks update-nodegroup-config --cluster-name "${CLUSTER}" --nodegroup-name "${NODEGROUP}" \
  --region "${REGION}" \
  --scaling-config "minSize=1,maxSize=4,desiredSize=${DESIRED_NODES}" >/dev/null 2>&1 || true

# ---- 7. Helm install (microservices) ----
echo ">> helm upgrade --install nexogent"
helm upgrade --install nexogent deploy/helm/nexogent \
  --namespace "${NAMESPACE}" --create-namespace \
  --set backend.image.repository="${BACKEND_IMAGE}" \
  --set backend.image.tag="${IMAGE_TAG}" \
  --set frontend.image.repository="${FRONTEND_IMAGE}" \
  --set frontend.image.tag="${IMAGE_TAG}" \
  --set serviceAccount.roleArn="${ROLE_ARN}" \
  --set region="${REGION}" \
  --wait --timeout 6m

echo ">> Done. Workloads:"
kubectl -n "${NAMESPACE}" get deploy,svc,cronjob,ingress,pods

URL="$(kubectl -n "${NAMESPACE}" get ingress nexogent -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || true)"
echo
if [ -n "${URL}" ]; then
  echo ">> Public URL (ALB may take ~2-3 min): http://${URL}"
else
  echo ">> Ingress ALB still provisioning. Re-check with:"
  echo "   kubectl -n ${NAMESPACE} get ingress nexogent -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'"
fi
echo ">> Trigger an immediate Prowler scan (otherwise it runs on schedule):"
echo "   kubectl -n ${NAMESPACE} create job --from=cronjob/nexogent-prowler prowler-now"
