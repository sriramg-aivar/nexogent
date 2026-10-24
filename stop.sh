#!/usr/bin/env bash
# Nexogent — teardown to stop billing.
# Uninstalls the Helm release and scales the node group down. Does NOT delete
# the cluster, the ECR repo, or the IAM policy/role (pass --purge for those).
set -euo pipefail

REGION="${REGION:-us-east-1}"
CLUSTER="${CLUSTER:-eks-lab}"
NODEGROUP="${NODEGROUP:-core}"
NAMESPACE="${NAMESPACE:-nexogent}"
REPO="${REPO:-nexogent}"
POLICY_NAME="${POLICY_NAME:-NexogentReadOnly}"
ROLE_NAME="${ROLE_NAME:-NexogentReadOnly}"
SA_NAME="${SA_NAME:-nexogent-sa}"

cd "$(dirname "$0")"
ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"

echo ">> Uninstalling Helm release"
helm uninstall nexogent --namespace "${NAMESPACE}" 2>/dev/null || true

echo ">> Scaling node group ${NODEGROUP} down to 0 desired"
aws eks update-nodegroup-config --cluster-name "${CLUSTER}" --nodegroup-name "${NODEGROUP}" \
  --region "${REGION}" \
  --scaling-config "minSize=0,maxSize=4,desiredSize=0" >/dev/null 2>&1 || true

if [[ "${1:-}" == "--purge" ]]; then
  echo ">> --purge: removing IRSA service account, IAM role/policy, ECR repo, namespace"
  eksctl delete iamserviceaccount --cluster "${CLUSTER}" --region "${REGION}" \
    --namespace "${NAMESPACE}" --name "${SA_NAME}" 2>/dev/null || true
  POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"
  aws iam detach-role-policy --role-name "${ROLE_NAME}" --policy-arn "${POLICY_ARN}" 2>/dev/null || true
  aws iam delete-policy --policy-arn "${POLICY_ARN}" 2>/dev/null || true
  aws ecr delete-repository --repository-name "${REPO}" --region "${REGION}" --force 2>/dev/null || true
  kubectl delete namespace "${NAMESPACE}" 2>/dev/null || true
  echo ">> Purge complete."
else
  echo ">> Release removed and nodes scaled to 0. Re-run start.sh to bring it back."
  echo ">> (Use: ./stop.sh --purge  to also delete IAM role/policy, ECR repo, namespace.)"
fi
