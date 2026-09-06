---
name: security-audit
description: Audit code for security vulnerabilities and risks
category: security
context: fork
agent: Explore
allowed-tools: "Read, Grep, Glob"
---
Perform a security audit on $ARGUMENTS:

1. Scan for common vulnerability patterns:
   - SQL injection / command injection
   - XSS and output encoding issues
   - Authentication and authorization flaws
   - Hardcoded secrets or credentials
   - Insecure deserialization
2. Check dependency versions for known CVEs
3. Review access control and permission boundaries
4. Identify sensitive data handling (PII, tokens, keys)
5. Provide a prioritized list of findings with remediation steps
