// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

import type {Metadata} from "next";
import {PushNotificationManager} from "@/components/common/admin/push-notification";

export const metadata: Metadata = {
  title: "通知推送 - Wavelet Admin",
  description: "系统通知多渠道推送与事件管理控制台",
};

export default function PushAdminPage() {
  return <PushNotificationManager />;
}
