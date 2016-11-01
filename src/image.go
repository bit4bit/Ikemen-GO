package main

import (
	"encoding/binary"
	"github.com/go-gl/gl/v2.1/gl"
	"os"
	"strings"
)

func gltest() {
	vertShader := strings.Join([]string{
		"void main(void){",
		"gl_TexCoord[0] = gl_TextureMatrix[0] * gl_MultiTexCoord0;",
		"gl_Position = ftransform();",
		"}\x00"}, "")
	fragShader := strings.Join([]string{
		"uniform float a;",
		"uniform sampler2D tex;",
		"uniform sampler1D pal;",
		"uniform int msk;",
		"void main(void){",
		"float r = texture2D(tex, gl_TexCoord[0].st).r;",
		"vec4 c;",
		"gl_FragColor =",
		"int(255.0*r) == msk ? vec4(0.0)",
		": (c = texture1D(pal, r*0.9961), vec4(c.b, c.g, c.r, a));",
		"}\x00"}, "")
	fragShaderFc := strings.Join([]string{
		"uniform float a;",
		"uniform sampler2D tex;",
		"uniform bool neg;",
		"uniform float gray;",
		"uniform vec3 add;",
		"uniform vec3 mul;",
		"void main(void){",
		"vec4 c = texture2D(tex, gl_TexCoord[0].st);",
		"if(neg) c.rgb = vec3(1.0) - c.rgb;",
		"float gcol = (c.r + c.g + c.b) / 3.0;",
		"c.r += (gcol - c.r) * gray + add.r;",
		"c.g += (gcol - c.g) * gray + add.g;",
		"c.b += (gcol - c.b) * gray + add.b;",
		"c.rgb *= mul;",
		"c.a *= a;",
		"gl_FragColor = c;",
		"}\x00"}, "")
	fragShaderFcS := strings.Join([]string{
		"uniform float a;",
		"uniform sampler2D tex;",
		"uniform vec3 color;",
		"void main(void){",
		"vec4 c = texture2D(tex, gl_TexCoord[0].st);",
		"c.rgb = color * c.a;",
		"c.a *= a;",
		"gl_FragColor = c;",
		"}\x00"}, "")
	errLog := func(obl uintptr) error {
		var size int32
		gl.GetObjectParameterivARB(obl, gl.INFO_LOG_LENGTH, &size)
		if size <= 0 {
			return nil
		}
		var l int32
		str := make([]byte, size+1)
		gl.GetInfoLogARB(obl, size, &l, &str[0])
		return Error(str[:l])
	}
	compile := func(shaderType uint32, src string) (shader uintptr) {
		shader = gl.CreateShaderObjectARB(shaderType)
		s, l := gl.Str(src), int32(len(src)-1)
		gl.ShaderSourceARB(shader, 1, &s, &l)
		gl.CompileShaderARB(shader)
		var ok int32
		gl.GetObjectParameterivARB(shader, gl.OBJECT_COMPILE_STATUS_ARB, &ok)
		if ok == 0 {
			chk(errLog(shader))
			panic(Error("コンパイルエラー"))
		}
		return
	}
	link := func(v uintptr, f uintptr) (program uintptr) {
		program = gl.CreateProgramObjectARB()
		gl.AttachObjectARB(program, v)
		gl.AttachObjectARB(program, f)
		gl.LinkProgramARB(program)
		var ok int32
		gl.GetObjectParameterivARB(program, gl.OBJECT_LINK_STATUS_ARB, &ok)
		if ok == 0 {
			chk(errLog(program))
			panic(Error("リンクエラー"))
		}
		return
	}
	vertObj := compile(gl.VERTEX_SHADER, vertShader)
	fragObj := compile(gl.FRAGMENT_SHADER, fragShader)
	shader := link(vertObj, fragObj)
	gl.GetUniformLocationARB(shader, gl.Str("pal\x00"))
	gl.GetUniformLocationARB(shader, gl.Str("msk\x00"))
	gl.DeleteObjectARB(fragObj)
	fragObj = compile(gl.FRAGMENT_SHADER, fragShaderFc)
	shaderFc := link(vertObj, fragObj)
	gl.GetUniformLocationARB(shaderFc, gl.Str("neg\x00"))
	gl.GetUniformLocationARB(shaderFc, gl.Str("gray\x00"))
	gl.GetUniformLocationARB(shaderFc, gl.Str("add\x00"))
	gl.GetUniformLocationARB(shaderFc, gl.Str("mul\x00"))
	gl.DeleteObjectARB(fragObj)
	fragObj = compile(gl.FRAGMENT_SHADER, fragShaderFcS)
	shaderFcS := link(vertObj, fragObj)
	gl.GetUniformLocationARB(shaderFcS, gl.Str("color\x00"))
	gl.DeleteObjectARB(fragObj)
	gl.DeleteObjectARB(vertObj)
}

type SffHeader struct {
	ver0, ver1, ver2, ver3   byte
	firstSpriteHeaderOffset  uint32
	firstPaletteHeaderOffset uint32
	numberOfPalettes         uint32
	numberOfSprites          uint32
}

func (sh *SffHeader) read(f *os.File, lofs *uint32, tofs *uint32) error {
	buf := make([]byte, 12)
	n, err := f.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "ElecbyteSpr\x00" {
		return Error("ElecbyteSprではありません")
	}
	read := func(x interface{}) error {
		return binary.Read(f, binary.LittleEndian, x)
	}
	if err := read(&sh.ver3); err != nil {
		return err
	}
	if err := read(&sh.ver2); err != nil {
		return err
	}
	if err := read(&sh.ver1); err != nil {
		return err
	}
	if err := read(&sh.ver0); err != nil {
		return err
	}
	var dummy uint32
	if err := read(&dummy); err != nil {
		return err
	}
	switch sh.ver0 {
	case 1:
		sh.firstPaletteHeaderOffset ,sh.numberOfPalettes = 0,0
		if err := read(&sh.numberOfSprites); err != nil {
			return err
		}
		if err := read(&sh.firstSpriteHeaderOffset); err != nil {
			return err
		}
		if err := read(&dummy); err != nil {
			return err
		}
	case 2:
		for i := 0; i < 4; i++ {
			if err := read(&dummy); err != nil {
				return err
			}
		}
		if err := read(&sh.firstSpriteHeaderOffset); err != nil {
			return err
		}
		if err := read(&sh.numberOfSprites); err != nil {
			return err
		}
		if err := read(&sh.firstPaletteHeaderOffset); err != nil {
			return err
		}
		if err := read(&sh.numberOfPalettes); err != nil {
			return err
		}
		if err := read(lofs); err != nil {
			return err
		}
		if err := read(&dummy); err != nil {
			return err
		}
		if err := read(tofs); err != nil {
			return err
		}
	default:
		return Error("バージョンが不正です")
	}
	return nil
}