package secureentry

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

const (
	cursorInterruptDuration = 300 * time.Millisecond
	cursorFadeAlpha         = uint8(0x16) // 22/255: very faint "faded" state
	cursorFadeRatio         = float32(0.2)

	fadeStart = 0.5 - cursorFadeRatio/2
	fadeStop  = 0.5 + cursorFadeRatio/2
)

// cursorAnim drives the blinking of the entry's text cursor. It mirrors the
// behaviour of the standard Fyne entry cursor.
type cursorAnim struct {
	cursor            *canvas.Rectangle
	anim              *fyne.Animation
	lastInterruptTime time.Time
}

func newCursorAnim(cursor *canvas.Rectangle) *cursorAnim {
	return &cursorAnim{cursor: cursor}
}

func (a *cursorAnim) createAnim() *fyne.Animation {
	n := color.NRGBAModel.Convert(a.primary()).(color.NRGBA)
	opaqueColor := n
	endColor := color.NRGBA{R: n.R, G: n.G, B: n.B, A: cursorFadeAlpha}
	startColor := opaqueColor
	a.cursor.FillColor = startColor

	deltaA := float32(int(endColor.A) - int(startColor.A))
	interrupted := false

	anim := fyne.NewAnimation(time.Second/2, func(f float32) {
		if time.Since(a.lastInterruptTime) < cursorInterruptDuration {
			if !interrupted {
				a.cursor.FillColor = opaqueColor
				a.cursor.Refresh()
				interrupted = true
			}
			return
		}
		if interrupted {
			interrupted = false
			// Stop and start effectively restarts the animation from the beginning.
			a.anim.Stop()
			a.anim.Start()
			return
		}

		var alpha uint8
		switch {
		case f < fadeStart:
			if a.cursor.FillColor == startColor {
				return
			}
			a.cursor.FillColor = startColor
		case f > fadeStop:
			if a.cursor.FillColor == endColor {
				return
			}
			a.cursor.FillColor = endColor
		default:
			fade := (f - fadeStart) / cursorFadeRatio
			alpha = startColor.A + uint8(deltaA*fade)
			a.cursor.FillColor = color.NRGBA{R: n.R, G: n.G, B: n.B, A: alpha}
		}
		a.cursor.Refresh()
	})

	anim.RepeatCount = fyne.AnimationRepeatForever
	anim.AutoReverse = true
	return anim
}

func (a *cursorAnim) start() {
	if a.anim == nil {
		a.anim = a.createAnim()
		a.anim.Start()
	}
}

// interrupt keeps the cursor solid for a short time after typing.
func (a *cursorAnim) interrupt() {
	a.lastInterruptTime = time.Now()
}

func (a *cursorAnim) stop() {
	if a.anim != nil {
		a.anim.Stop()
		a.anim = nil
	}
}

func (a *cursorAnim) primary() color.Color {
	return themePrimaryColor()
}