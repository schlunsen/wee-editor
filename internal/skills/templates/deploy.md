---
name: deploy
description: Deploy the application to a target environment with safety checks
category: devops
disable-model-invocation: true
argument-hint: "[environment]"
---
Deploy the application to $0 environment:

1. Run the test suite to ensure all tests pass
2. Build the application for production
3. Push to the deployment target
4. Verify the deployment was successful
5. Report the deployment status
