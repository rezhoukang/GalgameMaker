"""用已有 4 秒 MP4，通过编辑器 API 创建并导出循环剧情。"""
import argparse
import base64
import html
import json
import shutil
import uuid
from pathlib import Path
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]
NAME = "循环车站 · 短视频 Demo"
SCENES = [
    ("01_clouds_4s.mp4", "天亮之前", "01 / 醒来", "05:17",
     "你在无人的车站醒来。时钟停在 05:17。\n一只黑猫正沿着站台走向河边。", "云层", False),
    ("02_cat_4s.mp4", "它记得你", "02 / 线索", "05:17",
     "猫停下来，望向你。\n车票背面浮出一个字：「归」。", "行走的黑猫", False),
    ("03_river_4s.mp4", "被折回的清晨", "03 / 回声", "05:17",
     "河面倒映着同一片天空。\n会重来的只有这一天，不是你的记忆。", "雾中的河流", False),
    ("04_traffic_4s.mp4", "这一次，走出去", "04 / 归途", "05:18",
     "你终于想起了回家的路。\n城市开始流动，清晨也开始向前。", "街道与车流", True),
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api", default="http://127.0.0.1:8787/api")
    args = parser.parse_args()

    def call(path, data=None, method=None, content_type="application/json"):
        body = json.dumps(data, ensure_ascii=False).encode() if isinstance(data, dict) else data
        req = Request(args.api.rstrip("/") + path, data=body,
                      headers={"Content-Type": content_type}, method=method or ("POST" if body else "GET"))
        with urlopen(req, timeout=60) as response:
            result = json.load(response)
        if result["code"] != 0:
            raise RuntimeError(result)
        return result["data"]

    def upload(scene_id, page):
        boundary = "demo-" + uuid.uuid4().hex
        body = (f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="scene.html"\r\n'
                'Content-Type: text/html; charset=utf-8\r\n\r\n').encode() + page
        body += f"\r\n--{boundary}--\r\n".encode()
        return call(f"/scenes/{scene_id}/html", body, "PUT", f"multipart/form-data; boundary={boundary}")

    # 先检查素材，再创建工程；重复运行不会覆盖已有作品。
    for filename, *_ in SCENES:
        if not (ROOT / "test_temp" / filename).is_file():
            raise FileNotFoundError(f"缺少素材：test_temp/{filename}")
    tree = call("/tree")
    if NAME in json.dumps(tree, ensure_ascii=False):
        raise RuntimeError("演示工程已经存在，请在编辑器中打开；本脚本不会覆盖已有工程。")
    template = Path(__file__).with_name("scene.html").read_text(encoding="utf-8")
    folder = call("/folders", {"parentId": None, "name": "演示工程"})
    project = call("/canvases", {"folderId": folder["id"], "name": NAME})
    project_id = project["id"]
    view = call(f"/canvas/{project_id}")
    start = view["scenes"][0]
    cat = call(f'/scenes/{start["id"]}/next', {})
    river = call(f'/scenes/{cat["scene"]["id"]}/next', {})
    ending = call(f'/scenes/{start["id"]}/next', {})
    loop = call(f'/scenes/{river["scene"]["id"]}/magic-port', {"hash": start["hash"]})
    call(f"/canvas/{project_id}/variables", {"variables": [
        {"name": "记忆", "initialValue": 0}, {"name": "体力", "initialValue": 2}
    ]}, "PUT")
    variables = {v["name"]: v["id"] for v in call(f"/canvas/{project_id}")["variables"]}

    def effect(name, value, set_value=0):
        return {"variableId": variables[name], "set": set_value, "delta": value}

    def condition(name, operator, value):
        return {"variableId": variables[name], "operator": operator, "value": value}

    def configure(port, label, conditions=None, effects=None):
        call(f'/nodes/{port["id"]}', {"name": label, "isShow": True, "rules": {
            "conditionMode": "all", "conditions": conditions or [], "effects": effects or []
        }}, "PUT")

    configure(cat["outNode"], "跟上黑猫，寻找线索", [condition("体力", ">=", 1)],
              [effect("记忆", 1), effect("体力", -1)])
    configure(river["outNode"], "带着车票，走向河岸")
    configure(loop["outNode"], "记住这个字，再醒一次", effects=[effect("体力", 2, 1)])
    configure(ending["outNode"], "想起归途，走出车站",
              [condition("记忆", ">=", 2), condition("体力", ">=", 1)])
    nodes = [start, cat["scene"], river["scene"], ending["scene"]]
    for node, (filename, title, chapter, clock, text, label, is_ending) in zip(nodes, SCENES):
        call(f'/scenes/{node["id"]}', {"name": title}, "PUT")
        replacements = {
            "__VIDEO__": base64.b64encode((ROOT / "test_temp" / filename).read_bytes()).decode(),
            "__TITLE__": html.escape(title), "__CHAPTER__": html.escape(chapter),
            "__CLOCK__": clock, "__TEXT__": html.escape(text), "__VIDEO_LABEL__": html.escape(label),
            "__ENDING__": '<div class="ending">END / 归途 · 谢谢你记得</div>' if is_ending else "",
        }
        page = template
        for key, value in replacements.items():
            page = page.replace(key, value)
        upload(node["id"], page.encode())
    issues = call(f"/canvas/{project_id}/check")
    if issues:
        raise RuntimeError(issues)
    exported = call("/export", {"projectId": project_id})
    output = Path(exported["outputDir"])
    # 仅为这个导出副本定制启动页；剧情、路由和变量仍由项目播放器执行。
    index = output / "index.html"
    page = index.read_text(encoding="utf-8").replace("Galgame 互动视频", "循环车站")
    page = page.replace('id="start-message">点击开始', 'id="start-message">每次醒来，都离归途近一点。')
    index.write_text(page, encoding="utf-8")
    destination = ROOT / "test_temp" / "loop-station-demo"
    if destination.exists():
        destination = destination.with_name(destination.name + "-" + str(project_id))
    shutil.copytree(output, destination)
    shutil.copyfile(ROOT / "test_temp" / "SOURCES.txt", destination / "视频来源.txt")
    shutil.copyfile(Path(__file__).with_name("README.md"), destination / "说明.md")
    archive = shutil.make_archive(str(destination), "zip", destination)
    manifest = {"projectId": project_id, "folderId": folder["id"], "name": NAME,
                "previewDir": str(destination), "exportDir": str(output), "zip": archive,
                "nodes": [n["id"] for n in nodes], "variables": variables}
    (ROOT / "test_temp" / "loop-station-manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
