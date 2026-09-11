# BundleTest demo video

The demo is a paced, narrated walkthrough of the local feedback loop, failure output,
changed-test selection, and cloud parity. It renders entirely with local tools, so it does
not upload the transcript or source code.

From `experimental/bundletest`:

```sh
uv run python demo/render_demo.py --output /tmp/bundletest-demo.mp4
```

The renderer needs Google Chrome and an FFmpeg build with the `flite` filter. Generate just
the voiceover when you want to edit it separately:

```sh
uv run python demo/render_demo.py --audio-only /tmp/bundletest-voiceover.wav
```

Use [`TRANSCRIPT.md`](TRANSCRIPT.md) to record the narration yourself or with another voice
generator. The scene headings are editing cues; only the quoted paragraphs are spoken.
