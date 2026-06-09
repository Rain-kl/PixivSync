/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package mail 提供 SMTP 邮件发送功能。
package mail

const (
	errDialTLSFailed            = "dial tls failed: %w"
	errSMTPClientCreationFailed = "smtp client creation failed: %w"
	errSMTPAuthFailed           = "smtp auth failed: %w"
	errSMTPMailCommandFailed    = "smtp mail command failed: %w"
	errSMTPRcptCommandFailed    = "smtp rcpt command failed: %w"
	errSMTPDataCommandFailed    = "smtp data command failed: %w"
	errSMTPWritingBodyFailed    = "smtp writing body failed: %w"
	errSendMailFailed           = "send mail failed: %w"
)
