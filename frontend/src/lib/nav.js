// 导航条目单一来源：侧栏与命令面板共用同一份视图清单（不发明第二份）。
import { Settings, Terminal } from "lucide-react";

import { GitMark } from "@/components/PlatformIcon";

export const NAV_ENTRIES = [
  { id: "repositories", label: "镜像仓库", icon: GitMark },
  { id: "logs", label: "运行日志", icon: Terminal },
  { id: "settings", label: "设置", icon: Settings },
];
