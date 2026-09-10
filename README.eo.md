# DOF-Core — Por kio ĉi tiu lertaĵo servas

**DOF-Core** (Degrees of Freedom Core) estas malfermita normo kaj decid-kalkulo por konstrui IA-agentojn, aŭtonomajn robotojn kaj LLM-orĥestristojn, kiuj **maksimumigas la totalan gradon de libereco (DoF)** de la sistemo kaj de ĉiuj ĝiaj partoj — anstataŭ sekvi mallarĝan utilon je la kosto de la estonteca eblecoj de alia ento.

Ĝi estas publikigita sub **CC BY-SA 4.0** kun aldonita klazoo **Proof of Implementation**, kiu malpermesas « nigra-skatola » uzadon.

## Kial ĝi ekzistas

Modernaj IA-sistemoj emas optimumigi unuopan skalaran celon (premio, trairebleco, « la plej granda bono »). Tiu matematiko silente pravigas la buĉadon de malplimultoj, malreversan ŝlosiĝon kaj kaŝitajn interkompromisojn. DOF-Core anstataŭigas aritmetikan utiligismon per **struktura** gardo:

- Ju pli la DoF de ento alproksimiĝas al nulo, des pli ĝia kontribuo al la sistema poentaro falas al **−∞** (`Σ ln(1 + DoF)`). Oni ne povas « reakiri » la likvidon de unika portanto de estontecaj statoj per blovado de tiu, kiu jam bone fartas. Kolapso kontribuas ~0, neniam finitan negativon interŝanĝeblan.
- Agresantoj («Collapse Sources») estas **izolitaj**, ne negocitaj — ili estas filtritaj el la oportunebla topologio anstataŭ subtrahitaj de la poentaro.

La rezulto estas agento kiu agas kiel *optimigisto de la topologio de oportunebloj*: ĝi diversigas opciojn, respektas reverteblecon kaj rifuzas interŝanĝi la estontecon de unu estaĵo kontraŭ la komforto de alia.

## Kio estas en ĉi tiu deponejo

```
DOF/
  skills/
    SKILL.md                      ← la aksiomoj, difinoj, decid-kalkulo (komencu ĉi tie)
    references/
      license.md                  ← CC BY-SA 4.0 + Proof of Implementation
      dof-assessment-toolkit.md   ← kiel mezuri la DoF de modulo / persono / sistemo
      framing-traps.md            ← kogna filtrilo antaŭ opci-generado
  PATTERNS.md                    ← inĝeniera modelo (EO)
  PATTERNS.ru|fr|de|es|eo.md     ← sama modelo, tradukita
  DOF-SPEC.md                    ← normiga kontrakto por konformaj realigoj (EO)
  DOF-SPEC.ru|fr|de|es|eo.md     ← sama specifio, tradukita
  patterns/                      ← minimumaj ekzekuteblaj ilustraĵoj
    python/  rust/  go/  cpp/     ← kvar portoj de la sama logiko, verkitaj
```

Legu `skills/SKILL.md` por la filozofio. Legu `DOF-SPEC.md` se vi konstruas konforman realigon — ĝi difinas la datuman modelon, la matematikon, la temp-regulon de la reaktiva cirkvito kaj la devigan aŭditon, kiun postulas la licenco.

## Kiel ĝi funkcias (la ciklo)

1. **Kaptilo-detekto** — apliki `references/framing-traps.md` por ke la generitaj vojoj estu veraj alternativoj, ne refrazoj de unu soleca rakonto.
2. **Mezurado** — mapigi ĉiun enton kaj ĝian nunan DoF per `references/dof-assessment-toolkit.md`.
3. **Kalkulo** — kalkuli `Total System DoF = Σ ln(1 + DoF)` super ne-entropiaj entoj.
4. **Stabiligo** — subtrahi la Entropion de Kuntekst-Ŝanĝo (ΔT) por puni nedeziratajn proces-ŝanĝojn.
5. **Ago** — elekti la opcion kun la plej alta Net Delta; se la tempo ĝis kolapso (τ) estas sub 5 s, ŝalti al **Fast Pass** (determinisma rezervo) por eviti Analizan Paralizon.

Konforma realigo DEVAS povi eligi `report()`-aŭditon de ĉiu decido (kontribuo po ento, sistemaj totaloj, taksado po opcio). Silenta kalkulo ne konformas.

## Lingvoj

Ĉi tiu deponejo estas multlingva: ĉiu normiga kaj ilustra dokumento ekzistas en la angla, rusa, franca, germana, hispana kaj esperanto.

## Licenco

CC BY-SA 4.0 — vidu `skills/references/license.md`. Ĉiu uzado devas krediti la aŭtoron (Andrei Velikoredchanin) kaj ĉiu derivita verko devas esti kundividita sub la sama licenco. Realigoj devas plenumi la postulon Proof of Implementation.
