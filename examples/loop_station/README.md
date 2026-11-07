# 循环车站

一个用 4 段 4 秒 MP4 制作的互动剧情示例。你被困在清晨的车站，跟随黑猫寻找线索；探索两轮后，新的出口出现。

## 游玩

打开导出目录中的 `index.html`，点击“新游戏”。视频结束后选择选项。可刷新后“继续游戏”。

路线：天亮之前 → 它记得你 → 被折回的清晨 → 回到天亮之前。两次探索后，选择“想起归途，走出车站”进入结局。

## 编辑器配置

- 工程：演示工程 / 循环车站 · 短视频 Demo。
- 变量：记忆初始为 0，体力初始为 2。
- 跟上黑猫：需要体力 ≥ 1；记忆 +1、体力 -1，`set=0`。
- 再醒一次：魔法端口返回起点；体力直接设为 2，`set=1`，记忆保留。
- 走出车站：同时满足记忆 ≥ 2、体力 ≥ 1 才显示。
- 视频内嵌在各节点的 HTML 中，并配有剧情文字；无需外部素材服务。
- 条件与效果分别保存在 `ports.conditions` 和 `ports.effects` 两个 JSON 字段，全部由编辑器导出的播放器执行。

## 重建示例

后端启动后，在仓库根目录运行：

```powershell
python examples/loop_station/create_demo.py
```

需要 `test_temp` 中的四段视频及 `SOURCES.txt`。脚本通过编辑器 API 创建新工程并导出，拒绝覆盖同名工程。导出副本和 ZIP 放在 `test_temp`；媒体和运行数据不提交 Git。

素材来自 Wikimedia Commons，原素材记录标注 CC0 1.0；每段截取前 4 秒。作者和来源：

- 云层：L. Shyamal — https://commons.wikimedia.org/wiki/File:Flight_over_clouds.webm
- 黑猫：Dzkouslavia — https://commons.wikimedia.org/wiki/File:Black_Cat_walking.webm
- 河流：Digitura — https://commons.wikimedia.org/wiki/File:Misty_river_47_seconds.webm
- 车流：137a — https://commons.wikimedia.org/wiki/File:Temporary_traffic_signal_at_Colonel_By_Drive.webm

详细转码信息保存在导出包的 `视频来源.txt`。剧情文字为本示例编写。
