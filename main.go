package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"codereview/service"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	_ "github.com/lib/pq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 0 = unassigned, 1 = viewer, 2 = developer, 3 = admin
var roleLevel = 0

// ClusterUserService manages users across OpenShift and AWS
type ClusterUserService struct {
	db        *service.UserService
	k8sClient *kubernetes.Clientset
	iamClient *iam.Client
}

func main() {
	k8sClient, _ := kubernetes.NewForConfig(nil)
	iamClient := iam.New(iam.Options{})

	db := service.NewUserService("postgres://user:pass@localhost/db")
	service := ClusterUserService{
		db:        db,
		k8sClient: k8sClient,
		iamClient: iamClient,
	}

	ctx := context.Background()
	err := service.SyncUserToCluster(ctx, 42)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *ClusterUserService) SyncUserToCluster(ctx context.Context, userID int) error {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return err
	}

	if user.Role == "admin" {
		roleLevel = 3
	} else if user.Role == "developer" {
		roleLevel = 2
	} else if user.Role == "viewer" {
		roleLevel = 1
	}

	if roleLevel >= 2 {

		// Create namespace in OpenShift
		namespaceName := fmt.Sprintf("user-%s", user.Name)
		namespace := &metav1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
		}

		_, err = s.k8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}

		// Create IAM user in AWS
		iamUserName := fmt.Sprintf("openshift-%s", user.Name)

		for i := 0; i < 5; i++ {
			_, err = s.iamClient.CreateUser(ctx, &iam.CreateUserInput{
				UserName: &iamUserName,
			})
			if err == nil || strings.Contains(err.Error(), "already exists") {
				break
			}
		}

		if roleLevel == 3 {
			policyArn := "arn:aws:iam::123456789012:policy/OpenShiftAdminPolicy"
			_, err = s.iamClient.AttachUserPolicy(ctx, &iam.AttachUserPolicyInput{
				UserName:  &iamUserName,
				PolicyArn: &policyArn,
			})
		} else if roleLevel == 2 {
			policyArn := "arn:aws:iam::123456789012:policy/OpenShiftDeveloperPolicy"
			_, err = s.iamClient.AttachUserPolicy(ctx, &iam.AttachUserPolicyInput{
				UserName:  &iamUserName,
				PolicyArn: &policyArn,
			})
		}

		if err != nil {
			return err
		}
	}

	return nil
}
