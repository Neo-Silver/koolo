package action

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/object"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/d2go/pkg/nip"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/event"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
	"github.com/lxn/win"
)

// StatInfo contains information about how to format a particular stat
type StatInfo struct {
	Format    string // e.g., "+%d to %s", "%d%% Better Chance of Getting Magic Items"
	UsePrefix bool   // whether to use + prefix for positive values
}

// StatMap contains the formatting rules for different stats
var StatMap = map[stat.ID]StatInfo{
	stat.Strength:                        {"%d Strength", true},
	stat.Energy:                          {"%d Energy", true},
	stat.Dexterity:                       {"%d Dexterity", true},
	stat.Vitality:                        {"%d Vitality", true},
	stat.Life:                            {"%d Life", true},
	stat.MaxLife:                         {"%d MaxLife", true},
	stat.Mana:                            {"%d Mana", true},
	stat.MaxMana:                         {"%d MaxMana", true},
	stat.Stamina:                         {"%d Stamina", true},
	stat.MaxStamina:                      {"%d MaxStamina", true},
	stat.Level:                           {"%d Level", true},
	stat.Experience:                      {"%d Experience", true},
	stat.Gold:                            {"%d Gold", true},
	stat.StashGold:                       {"%d StashGold", true},
	stat.EnhancedDefense:                 {"%d EnhancedDefense", true},
	stat.EnhancedDamageMin:               {"%d EnhancedDamageMin", true},
	stat.EnhancedDamage:                  {"%d EnhancedDamage", true},
	stat.AttackRating:                    {"%d AttackRating", true},
	stat.ChanceToBlock:                   {"%d ChanceToBlock", true},
	stat.MinDamage:                       {"%d MinDamage", true},
	stat.MaxDamage:                       {"%d MaxDamage", true},
	stat.TwoHandedMinDamage:              {"%d TwoHandedMinDamage", true},
	stat.TwoHandedMaxDamage:              {"%d TwoHandedMaxDamage", true},
	stat.DamagePercent:                   {"%d DamagePercent", true},
	stat.ManaRecovery:                    {"%d ManaRecovery", true},
	stat.ManaRecoveryBonus:               {"%d ManaRecoveryBonus", true},
	stat.StaminaRecoveryBonus:            {"%d StaminaRecoveryBonus", true},
	stat.LastExp:                         {"%d LastExp", true},
	stat.NextExp:                         {"%d NextExp", true},
	stat.Defense:                         {"%d Defense", true},
	stat.DefenseVsMissiles:               {"%d DefenseVsMissiles", true},
	stat.DefenseVsHth:                    {"%d DefenseVsHth", true},
	stat.NormalDamageReduction:           {"%d NormalDamageReduction", true},
	stat.MagicDamageReduction:            {"%d MagicDamageReduction", true},
	stat.DamageReduced:                   {"%d DamageReduced", true},
	stat.MagicResist:                     {"%d MagicResist", true},
	stat.MaxMagicResist:                  {"%d MaxMagicResist", true},
	stat.FireResist:                      {"%d FireResist", true},
	stat.MaxFireResist:                   {"%d MaxFireResist", true},
	stat.LightningResist:                 {"%d LightningResist", true},
	stat.MaxLightningResist:              {"%d MaxLightningResist", true},
	stat.ColdResist:                      {"%d ColdResist", true},
	stat.MaxColdResist:                   {"%d MaxColdResist", true},
	stat.PoisonResist:                    {"%d PoisonResist", true},
	stat.MaxPoisonResist:                 {"%d MaxPoisonResist", true},
	stat.DamageAura:                      {"%d DamageAura", true},
	stat.FireMinDamage:                   {"%d FireMinDamage", true},
	stat.FireMaxDamage:                   {"%d FireMaxDamage", true},
	stat.LightningMinDamage:              {"%d LightningMinDamage", true},
	stat.LightningMaxDamage:              {"%d LightningMaxDamage", true},
	stat.MagicMinDamage:                  {"%d MagicMinDamage", true},
	stat.MagicMaxDamage:                  {"%d MagicMaxDamage", true},
	stat.ColdMinDamage:                   {"%d ColdMinDamage", true},
	stat.ColdMaxDamage:                   {"%d ColdMaxDamage", true},
	stat.ColdLength:                      {"%d ColdLength", true},
	stat.PoisonMinDamage:                 {"%d PoisonMinDamage", true},
	stat.PoisonMaxDamage:                 {"%d PoisonMaxDamage", true},
	stat.PoisonLength:                    {"%d PoisonLength", true},
	stat.LifeSteal:                       {"%d LifeSteal", true},
	stat.LifeStealMax:                    {"%d LifeStealMax", true},
	stat.ManaSteal:                       {"%d ManaSteal", true},
	stat.ManaStealMax:                    {"%d ManaStealMax", true},
	stat.StaminaDrainMinDamage:           {"%d StaminaDrainMinDamage", true},
	stat.StaminaDrainMaxDamage:           {"%d StaminaDrainMaxDamage", true},
	stat.StunLength:                      {"%d StunLength", true},
	stat.VelocityPercent:                 {"%d VelocityPercent", true},
	stat.AttackRate:                      {"%d AttackRate", true},
	stat.OtherAnimRate:                   {"%d OtherAnimRate", true},
	stat.Quantity:                        {"%d Quantity", true},
	stat.Value:                           {"%d Value", true},
	stat.Durability:                      {"%d Durability", true},
	stat.MaxDurability:                   {"%d MaxDurability", true},
	stat.ReplenishLife:                   {"%d ReplenishLife", true},
	stat.MaxDurabilityPercent:            {"%d MaxDurabilityPercent", true},
	stat.MaxLifePercent:                  {"%d MaxLifePercent", true},
	stat.MaxManaPercent:                  {"%d MaxManaPercent", true},
	stat.AttackerTakesDamage:             {"%d AttackerTakesDamage", true},
	stat.GoldFind:                        {"%d GoldFind", true},
	stat.MagicFind:                       {"%d MagicFind", true},
	stat.Knockback:                       {"%d Knockback", true},
	stat.TimeDuration:                    {"%d TimeDuration", true},
	stat.AddClassSkills:                  {"%d AddClassSkills", true},
	stat.AddExperience:                   {"%d AddExperience", true},
	stat.LifeAfterEachKill:               {"%d LifeAfterEachKill", true},
	stat.ReducePrices:                    {"%d ReducePrices", true},
	stat.DoubleHerbDuration:              {"%d DoubleHerbDuration", true},
	stat.LightRadius:                     {"%d LightRadius", true},
	stat.LightColor:                      {"%d LightColor", true},
	stat.Requirements:                    {"%d Requirements", true},
	stat.LevelRequire:                    {"%d LevelRequire", true},
	stat.IncreasedAttackSpeed:            {"%d IncreasedAttackSpeed", true},
	stat.LevelRequirePercent:             {"%d LevelRequirePercent", true},
	stat.LastBlockFrame:                  {"%d LastBlockFrame", true},
	stat.FasterRunWalk:                   {"%d FasterRunWalk", true},
	stat.NonClassSkill:                   {"%d NonClassSkill", true},
	stat.State:                           {"%d State", true},
	stat.FasterHitRecovery:               {"%d FasterHitRecovery", true},
	stat.PlayerCount:                     {"%d PlayerCount", true},
	stat.PoisonOverrideLength:            {"%d PoisonOverrideLength", true},
	stat.FasterBlockRate:                 {"%d FasterBlockRate", true},
	stat.BypassUndead:                    {"%d BypassUndead", true},
	stat.BypassDemons:                    {"%d BypassDemons", true},
	stat.FasterCastRate:                  {"%d FasterCastRate", true},
	stat.BypassBeasts:                    {"%d BypassBeasts", true},
	stat.SingleSkill:                     {"%d SingleSkill", true},
	stat.SlainMonstersRestInPeace:        {"%d SlainMonstersRestInPeace", true},
	stat.CurseResistance:                 {"%d CurseResistance", true},
	stat.PoisonLengthReduced:             {"%d PoisonLengthReduced", true},
	stat.NormalDamage:                    {"%d NormalDamage", true},
	stat.HitCausesMonsterToFlee:          {"%d HitCausesMonsterToFlee", true},
	stat.HitBlindsTarget:                 {"%d HitBlindsTarget", true},
	stat.DamageTakenGoesToMana:           {"%d DamageTakenGoesToMana", true},
	stat.IgnoreTargetsDefense:            {"%d IgnoreTargetsDefense", true},
	stat.TargetDefense:                   {"%d TargetDefense", true},
	stat.PreventMonsterHeal:              {"%d PreventMonsterHeal", true},
	stat.HalfFreezeDuration:              {"%d HalfFreezeDuration", true},
	stat.AttackRatingPercent:             {"%d AttackRatingPercent", true},
	stat.MonsterDefensePerHit:            {"%d MonsterDefensePerHit", true},
	stat.DemonDamagePercent:              {"%d DemonDamagePercent", true},
	stat.UndeadDamagePercent:             {"%d UndeadDamagePercent", true},
	stat.DemonAttackRating:               {"%d DemonAttackRating", true},
	stat.UndeadAttackRating:              {"%d UndeadAttackRating", true},
	stat.Throwable:                       {"%d Throwable", true},
	stat.FireSkills:                      {"%d FireSkills", true},
	stat.AllSkills:                       {"%d AllSkills", true},
	stat.AttackerTakesLightDamage:        {"%d AttackerTakesLightDamage", true},
	stat.IronMaidenLevel:                 {"%d IronMaidenLevel", true},
	stat.LifeTapLevel:                    {"%d LifeTapLevel", true},
	stat.ThornsPercent:                   {"%d ThornsPercent", true},
	stat.BoneArmor:                       {"%d BoneArmor", true},
	stat.BoneArmorMax:                    {"%d BoneArmorMax", true},
	stat.FreezesTarget:                   {"%d FreezesTarget", true},
	stat.OpenWounds:                      {"%d OpenWounds", true},
	stat.CrushingBlow:                    {"%d CrushingBlow", true},
	stat.KickDamage:                      {"%d KickDamage", true},
	stat.ManaAfterKill:                   {"%d ManaAfterKill", true},
	stat.HealAfterDemonKill:              {"%d HealAfterDemonKill", true},
	stat.ExtraBlood:                      {"%d ExtraBlood", true},
	stat.DeadlyStrike:                    {"%d DeadlyStrike", true},
	stat.AbsorbFirePercent:               {"%d AbsorbFirePercent", true},
	stat.AbsorbFire:                      {"%d AbsorbFire", true},
	stat.AbsorbLightningPercent:          {"%d AbsorbLightningPercent", true},
	stat.AbsorbLightning:                 {"%d AbsorbLightning", true},
	stat.AbsorbMagicPercent:              {"%d AbsorbMagicPercent", true},
	stat.AbsorbMagic:                     {"%d AbsorbMagic", true},
	stat.AbsorbColdPercent:               {"%d AbsorbColdPercent", true},
	stat.AbsorbCold:                      {"%d AbsorbCold", true},
	stat.SlowsTarget:                     {"%d SlowsTarget", true},
	stat.Aura:                            {"%d Aura", true},
	stat.Indestructible:                  {"%d Indestructible", true},
	stat.CannotBeFrozen:                  {"%d CannotBeFrozen", true},
	stat.SlowerStaminaDrain:              {"%d SlowerStaminaDrain", true},
	stat.Reanimate:                       {"%d Reanimate", true},
	stat.Pierce:                          {"%d Pierce", true},
	stat.MagicArrow:                      {"%d MagicArrow", true},
	stat.ExplosiveArrow:                  {"%d ExplosiveArrow", true},
	stat.ThrowMinDamage:                  {"%d ThrowMinDamage", true},
	stat.ThrowMaxDamage:                  {"%d ThrowMaxDamage", true},
	stat.SkillHandofAthena:               {"%d SkillHandofAthena", true},
	stat.SkillStaminaPercent:             {"%d SkillStaminaPercent", true},
	stat.SkillPassiveStaminaPercent:      {"%d SkillPassiveStaminaPercent", true},
	stat.SkillConcentration:              {"%d SkillConcentration", true},
	stat.SkillEnchant:                    {"%d SkillEnchant", true},
	stat.SkillPierce:                     {"%d SkillPierce", true},
	stat.SkillConviction:                 {"%d SkillConviction", true},
	stat.SkillChillingArmor:              {"%d SkillChillingArmor", true},
	stat.SkillFrenzy:                     {"%d SkillFrenzy", true},
	stat.SkillDecrepify:                  {"%d SkillDecrepify", true},
	stat.SkillArmorPercent:               {"%d SkillArmorPercent", true},
	stat.Alignment:                       {"%d Alignment", true},
	stat.Target0:                         {"%d Target0", true},
	stat.Target1:                         {"%d Target1", true},
	stat.GoldLost:                        {"%d GoldLost", true},
	stat.ConverisonLevel:                 {"%d ConverisonLevel", true},
	stat.ConverisonMaxHP:                 {"%d ConverisonMaxHP", true},
	stat.UnitDooverlay:                   {"%d UnitDooverlay", true},
	stat.AttackVsMonType:                 {"%d AttackVsMonType", true},
	stat.DamageVsMonType:                 {"%d DamageVsMonType", true},
	stat.Fade:                            {"%d Fade", true},
	stat.ArmorOverridePercent:            {"%d ArmorOverridePercent", true},
	stat.AddSkillTab:                     {"%d AddSkillTab", true},
	stat.NumSockets:                      {"%d NumSockets", true},
	stat.SkillOnAttack:                   {"%d SkillOnAttack", true},
	stat.SkillOnKill:                     {"%d SkillOnKill", true},
	stat.SkillOnDeath:                    {"%d SkillOnDeath", true},
	stat.SkillOnHit:                      {"%d SkillOnHit", true},
	stat.SkillOnLevelUp:                  {"%d SkillOnLevelUp", true},
	stat.SkillOnGetHit:                   {"%d SkillOnGetHit", true},
	stat.ItemChargedSkill:                {"%d ItemChargedSkill", true},
	stat.DefensePerLevel:                 {"%d DefensePerLevel", true},
	stat.ArmorPercentPerLevel:            {"%d ArmorPercentPerLevel", true},
	stat.LifePerLevel:                    {"%d LifePerLevel", true},
	stat.ManaPerLevel:                    {"%d ManaPerLevel", true},
	stat.MaxDamagePerLevel:               {"%d MaxDamagePerLevel", true},
	stat.MaxDamagePercentPerLevel:        {"%d MaxDamagePercentPerLevel", true},
	stat.StrengthPerLevel:                {"%d StrengthPerLevel", true},
	stat.DexterityPerLevel:               {"%d DexterityPerLevel", true},
	stat.EnergyPerLevel:                  {"%d EnergyPerLevel", true},
	stat.VitalityPerLevel:                {"%d VitalityPerLevel", true},
	stat.AttackRatingPerLevel:            {"%d AttackRatingPerLevel", true},
	stat.AttackRatingPercentPerLevel:     {"%d AttackRatingPercentPerLevel", true},
	stat.ColdDamageMaxPerLevel:           {"%d ColdDamageMaxPerLevel", true},
	stat.FireDamageMaxPerLevel:           {"%d FireDamageMaxPerLevel", true},
	stat.LightningDamageMaxPerLevel:      {"%d LightningDamageMaxPerLevel", true},
	stat.PoisonDamageMaxPerLevel:         {"%d PoisonDamageMaxPerLevel", true},
	stat.ResistColdPerLevel:              {"%d ResistColdPerLevel", true},
	stat.ResistFirePerLevel:              {"%d ResistFirePerLevel", true},
	stat.ResistLightningPerLevel:         {"%d ResistLightningPerLevel", true},
	stat.ResistPoisonPerLevel:            {"%d ResistPoisonPerLevel", true},
	stat.AbsorbColdPerLevel:              {"%d AbsorbColdPerLevel", true},
	stat.AbsorbFirePerLevel:              {"%d AbsorbFirePerLevel", true},
	stat.AbsorbLightningPerLevel:         {"%d AbsorbLightningPerLevel", true},
	stat.AbsorbPoisonPerLevel:            {"%d AbsorbPoisonPerLevel", true},
	stat.ThornsPerLevel:                  {"%d ThornsPerLevel", true},
	stat.ExtraGoldPerLevel:               {"%d ExtraGoldPerLevel", true},
	stat.MagicFindPerLevel:               {"%d MagicFindPerLevel", true},
	stat.RegenStaminaPerLevel:            {"%d RegenStaminaPerLevel", true},
	stat.StaminaPerLevel:                 {"%d StaminaPerLevel", true},
	stat.DamageDemonPerLevel:             {"%d DamageDemonPerLevel", true},
	stat.DamageUndeadPerLevel:            {"%d DamageUndeadPerLevel", true},
	stat.AttackRatingDemonPerLevel:       {"%d AttackRatingDemonPerLevel", true},
	stat.AttackRatingUndeadPerLevel:      {"%d AttackRatingUndeadPerLevel", true},
	stat.CrushingBlowPerLevel:            {"%d CrushingBlowPerLevel", true},
	stat.OpenWoundsPerLevel:              {"%d OpenWoundsPerLevel", true},
	stat.KickDamagePerLevel:              {"%d KickDamagePerLevel", true},
	stat.DeadlyStrikePerLevel:            {"%d DeadlyStrikePerLevel", true},
	stat.FindGemsPerLevel:                {"%d FindGemsPerLevel", true},
	stat.ReplenishDurability:             {"%d ReplenishDurability", true},
	stat.ReplenishQuantity:               {"%d ReplenishQuantity", true},
	stat.ExtraStack:                      {"%d ExtraStack", true},
	stat.FindItem:                        {"%d FindItem", true},
	stat.SlashDamage:                     {"%d SlashDamage", true},
	stat.SlashDamagePercent:              {"%d SlashDamagePercent", true},
	stat.CrushDamage:                     {"%d CrushDamage", true},
	stat.CrushDamagePercent:              {"%d CrushDamagePercent", true},
	stat.ThrustDamage:                    {"%d ThrustDamage", true},
	stat.ThrustDamagePercent:             {"%d ThrustDamagePercent", true},
	stat.AbsorbSlash:                     {"%d AbsorbSlash", true},
	stat.AbsorbCrush:                     {"%d AbsorbCrush", true},
	stat.AbsorbThrust:                    {"%d AbsorbThrust", true},
	stat.AbsorbSlashPercent:              {"%d AbsorbSlashPercent", true},
	stat.AbsorbCrushPercent:              {"%d AbsorbCrushPercent", true},
	stat.AbsorbThrustPercent:             {"%d AbsorbThrustPercent", true},
	stat.ArmorByTime:                     {"%d ArmorByTime", true},
	stat.ArmorPercentByTime:              {"%d ArmorPercentByTime", true},
	stat.LifeByTime:                      {"%d LifeByTime", true},
	stat.ManaByTime:                      {"%d ManaByTime", true},
	stat.MaxDamageByTime:                 {"%d MaxDamageByTime", true},
	stat.MaxDamagePercentByTime:          {"%d MaxDamagePercentByTime", true},
	stat.StrengthByTime:                  {"%d StrengthByTime", true},
	stat.DexterityByTime:                 {"%d DexterityByTime", true},
	stat.EnergyByTime:                    {"%d EnergyByTime", true},
	stat.VitalityByTime:                  {"%d VitalityByTime", true},
	stat.AttackRatingByTime:              {"%d AttackRatingByTime", true},
	stat.AttackRatingPercentByTime:       {"%d AttackRatingPercentByTime", true},
	stat.ColdDamageMaxByTime:             {"%d ColdDamageMaxByTime", true},
	stat.FireDamageMaxByTime:             {"%d FireDamageMaxByTime", true},
	stat.LightningDamageMaxByTime:        {"%d LightningDamageMaxByTime", true},
	stat.PoisonDamageMaxByTime:           {"%d PoisonDamageMaxByTime", true},
	stat.ResistColdByTime:                {"%d ResistColdByTime", true},
	stat.ResistFireByTime:                {"%d ResistFireByTime", true},
	stat.ResistLightningByTime:           {"%d ResistLightningByTime", true},
	stat.ResistPoisonByTime:              {"%d ResistPoisonByTime", true},
	stat.AbsorbColdByTime:                {"%d AbsorbColdByTime", true},
	stat.AbsorbFireByTime:                {"%d AbsorbFireByTime", true},
	stat.AbsorbLightningByTime:           {"%d AbsorbLightningByTime", true},
	stat.AbsorbPoisonByTime:              {"%d AbsorbPoisonByTime", true},
	stat.FindGoldByTime:                  {"%d FindGoldByTime", true},
	stat.MagicFindByTime:                 {"%d MagicFindByTime", true},
	stat.RegenStaminaByTime:              {"%d RegenStaminaByTime", true},
	stat.StaminaByTime:                   {"%d StaminaByTime", true},
	stat.DamageDemonByTime:               {"%d DamageDemonByTime", true},
	stat.DamageUndeadByTime:              {"%d DamageUndeadByTime", true},
	stat.AttackRatingDemonByTime:         {"%d AttackRatingDemonByTime", true},
	stat.AttackRatingUndeadByTime:        {"%d AttackRatingUndeadByTime", true},
	stat.CrushingBlowByTime:              {"%d CrushingBlowByTime", true},
	stat.OpenWoundsByTime:                {"%d OpenWoundsByTime", true},
	stat.KickDamageByTime:                {"%d KickDamageByTime", true},
	stat.DeadlyStrikeByTime:              {"%d DeadlyStrikeByTime", true},
	stat.FindGemsByTime:                  {"%d FindGemsByTime", true},
	stat.PierceCold:                      {"%d PierceCold", true},
	stat.PierceFire:                      {"%d PierceFire", true},
	stat.PierceLightning:                 {"%d PierceLightning", true},
	stat.PiercePoison:                    {"%d PiercePoison", true},
	stat.DamageVsMonster:                 {"%d DamageVsMonster", true},
	stat.DamagePercentVsMonster:          {"%d DamagePercentVsMonster", true},
	stat.AttackRatingVsMonster:           {"%d AttackRatingVsMonster", true},
	stat.AttackRatingPercentVsMonster:    {"%d AttackRatingPercentVsMonster", true},
	stat.AcVsMonster:                     {"%d AcVsMonster", true},
	stat.AcPercentVsMonster:              {"%d AcPercentVsMonster", true},
	stat.FireLength:                      {"%d FireLength", true},
	stat.BurningMin:                      {"%d BurningMin", true},
	stat.BurningMax:                      {"%d BurningMax", true},
	stat.ProgressiveDamage:               {"%d ProgressiveDamage", true},
	stat.ProgressiveSteal:                {"%d ProgressiveSteal", true},
	stat.ProgressiveOther:                {"%d ProgressiveOther", true},
	stat.ProgressiveFire:                 {"%d ProgressiveFire", true},
	stat.ProgressiveCold:                 {"%d ProgressiveCold", true},
	stat.ProgressiveLightning:            {"%d ProgressiveLightning", true},
	stat.ExtraCharges:                    {"%d ExtraCharges", true},
	stat.ProgressiveAttackRating:         {"%d ProgressiveAttackRating", true},
	stat.PoisonCount:                     {"%d PoisonCount", true},
	stat.DamageFrameRate:                 {"%d DamageFrameRate", true},
	stat.PierceIdx:                       {"%d PierceIdx", true},
	stat.FireSkillDamage:                 {"%d FireSkillDamage", true},
	stat.LightningSkillDamage:            {"%d LightningSkillDamage", true},
	stat.ColdSkillDamage:                 {"%d ColdSkillDamage", true},
	stat.PoisonSkillDamage:               {"%d PoisonSkillDamage", true},
	stat.EnemyFireResist:                 {"%d EnemyFireResist", true},
	stat.EnemyLightningResist:            {"%d EnemyLightningResist", true},
	stat.EnemyColdResist:                 {"%d EnemyColdResist", true},
	stat.EnemyPoisonResist:               {"%d EnemyPoisonResist", true},
	stat.PassiveCriticalStrike:           {"%d PassiveCriticalStrike", true},
	stat.PassiveDodge:                    {"%d PassiveDodge", true},
	stat.PassiveAvoid:                    {"%d PassiveAvoid", true},
	stat.PassiveEvade:                    {"%d PassiveEvade", true},
	stat.PassiveWarmth:                   {"%d PassiveWarmth", true},
	stat.PassiveMasteryMeleeAttackRating: {"%d PassiveMasteryMeleeAttackRating", true},
	stat.PassiveMasteryMeleeDamage:       {"%d PassiveMasteryMeleeDamage", true},
	stat.PassiveMasteryMeleeCritical:     {"%d PassiveMasteryMeleeCritical", true},
	stat.PassiveMasteryThrowAttackRating: {"%d PassiveMasteryThrowAttackRating", true},
	stat.PassiveMasteryThrowDamage:       {"%d PassiveMasteryThrowDamage", true},
	stat.PassiveMasteryThrowCritical:     {"%d PassiveMasteryThrowCritical", true},
	stat.PassiveWeaponBlock:              {"%d PassiveWeaponBlock", true},
	stat.SummonResist:                    {"%d SummonResist", true},
	stat.ModifierListSkill:               {"%d ModifierListSkill", true},
	stat.ModifierListLevel:               {"%d ModifierListLevel", true},
	stat.LastSentHPPercent:               {"%d LastSentHPPercent", true},
	stat.SourceUnitType:                  {"%d SourceUnitType", true},
	stat.SourceUnitID:                    {"%d SourceUnitID", true},
	stat.ShortParam1:                     {"%d ShortParam1", true},
	stat.QuestItemDifficulty:             {"%d QuestItemDifficulty", true},
	stat.PassiveMagicMastery:             {"%d PassiveMagicMastery", true},
	stat.PassiveMagicPierce:              {"%d PassiveMagicPierce", true},
	stat.SkillCooldown:                   {"%d SkillCooldown", true},
	stat.SkillMissileDamageScale:         {"%d SkillMissileDamageScale", true},
}

const (
	maxGoldPerStashTab = 2500000
)

func Stash(forceStash bool) error {
	ctx := context.Get()
	ctx.SetLastAction("Stash")

	ctx.Logger.Debug("Checking for items to stash...")
	if !isStashingRequired(forceStash) {
		return nil
	}

	ctx.Logger.Info("Stashing items...")

	switch ctx.Data.PlayerUnit.Area {
	case area.KurastDocks:
		MoveToCoords(data.Position{X: 5146, Y: 5067})
	case area.LutGholein:
		MoveToCoords(data.Position{X: 5130, Y: 5086})
	}

	bank, _ := ctx.Data.Objects.FindOne(object.Bank)
	InteractObject(bank,
		func() bool {
			return ctx.Data.OpenMenus.Stash
		},
	)

	stashGold()
	orderInventoryPotions()
	stashInventory(forceStash)
	step.CloseAllMenus()

	return nil
}

func orderInventoryPotions() {
	ctx := context.Get()
	ctx.SetLastStep("orderInventoryPotions")

	for _, i := range ctx.Data.Inventory.ByLocation(item.LocationInventory) {
		if i.IsPotion() {
			if ctx.CharacterCfg.Inventory.InventoryLock[i.Position.Y][i.Position.X] == 0 {
				continue
			}

			screenPos := ui.GetScreenCoordsForItem(i)
			utils.Sleep(100)
			ctx.HID.Click(game.RightButton, screenPos.X, screenPos.Y)
			utils.Sleep(200)
		}
	}
}

func isStashingRequired(firstRun bool) bool {
	ctx := context.Get()
	ctx.SetLastStep("isStashingRequired")

	for _, i := range ctx.Data.Inventory.ByLocation(item.LocationInventory) {
		stashIt, _, _ := shouldStashIt(i, firstRun)
		if stashIt {
			return true
		}
	}

	isStashFull := true
	for _, goldInStash := range ctx.Data.Inventory.StashedGold {
		if goldInStash < maxGoldPerStashTab {
			isStashFull = false
		}
	}

	if ctx.Data.Inventory.Gold > ctx.Data.PlayerUnit.MaxGold()/1 && !isStashFull {
		return true
	}

	return false
}

func stashGold() {
	ctx := context.Get()
	ctx.SetLastAction("stashGold")

	if ctx.Data.Inventory.Gold == 0 {
		return
	}

	ctx.Logger.Info("Stashing gold...", slog.Int("gold", ctx.Data.Inventory.Gold))

	for tab, goldInStash := range ctx.Data.Inventory.StashedGold {
		ctx.RefreshGameData()
		if ctx.Data.Inventory.Gold == 0 {
			return
		}

		if goldInStash < maxGoldPerStashTab {
			SwitchStashTab(tab + 1)
			clickStashGoldBtn()
			utils.Sleep(500)
		}
	}

	ctx.Logger.Info("All stash tabs are full of gold :D")
}

func stashInventory(firstRun bool) {
	ctx := context.Get()
	ctx.SetLastAction("stashInventory")

	currentTab := 1
	if ctx.CharacterCfg.Character.StashToShared {
		currentTab = 2
	}
	SwitchStashTab(currentTab)

	for _, i := range ctx.Data.Inventory.ByLocation(item.LocationInventory) {
		stashIt, matchedRule, ruleFile := shouldStashIt(i, firstRun)

		if !stashIt {
			continue
		}
		for currentTab < 5 {
			if stashItemAction(i, matchedRule, ruleFile, firstRun) {
				r, res := ctx.CharacterCfg.Runtime.Rules.EvaluateAll(i)

				if res != nip.RuleResultFullMatch && firstRun {
					ctx.Logger.Info(
						fmt.Sprintf("Item %s [%s] stashed because it was found in the inventory during the first run.", i.Desc().Name, i.Quality.ToString()),
					)
					break
				}

				ctx.Logger.Info(
					fmt.Sprintf("Item %s [%s] stashed", i.Desc().Name, i.Quality.ToString()),
					slog.String("nipFile", fmt.Sprintf("%s:%d", r.Filename, r.LineNumber)),
					slog.String("rawRule", r.RawLine),
				)
				break
			}
			if currentTab == 5 {
				ctx.Logger.Info("Stash is full ...")
				//TODO: Stash is full stop the bot
			}
			ctx.Logger.Debug(fmt.Sprintf("Tab %d is full, switching to next one", currentTab))
			currentTab++
			SwitchStashTab(currentTab)
		}
	}
}

func shouldStashIt(i data.Item, firstRun bool) (bool, string, string) {
	ctx := context.Get()
	ctx.SetLastStep("shouldStashIt")

	// Don't stash items from quests during leveling process, it makes things easier to track
	if _, isLevelingChar := ctx.Char.(context.LevelingCharacter); isLevelingChar && i.IsFromQuest() {
		return false, "", ""
	}

	if i.IsRuneword {
		return true, "Runeword", ""
	}

	// Stash items that are part of a recipe which are not covered by the NIP rules
	if shouldKeepRecipeItem(i) {
		return true, "Item is part of a enabled recipe", ""
	}

	// Don't stash the Tomes, keys and WirtsLeg
	if i.Name == item.TomeOfTownPortal || i.Name == item.TomeOfIdentify || i.Name == item.Key || i.Name == "WirtsLeg" {
		return false, "", ""
	}

	if i.Position.Y >= len(ctx.CharacterCfg.Inventory.InventoryLock) || i.Position.X >= len(ctx.CharacterCfg.Inventory.InventoryLock[0]) {
		return false, "", ""
	}

	if i.Location.LocationType == item.LocationInventory && ctx.CharacterCfg.Inventory.InventoryLock[i.Position.Y][i.Position.X] == 0 || i.IsPotion() {
		return false, "", ""
	}

	// Let's stash everything during first run, we don't want to sell items from the user
	if firstRun {
		return true, "FirstRun", ""
	}

	rule, res := ctx.CharacterCfg.Runtime.Rules.EvaluateAll(i)
	if res == nip.RuleResultFullMatch && doesExceedQuantity(rule) {
		return false, "", ""
	}

	// Full rule match
	if res == nip.RuleResultFullMatch {
		return true, rule.RawLine, rule.Filename + ":" + strconv.Itoa(rule.LineNumber)
	}
	return false, "", ""
}

func shouldKeepRecipeItem(i data.Item) bool {
	ctx := context.Get()
	ctx.SetLastStep("shouldKeepRecipeItem")

	// No items with quality higher than magic can be part of a recipe
	if i.Quality > item.QualityMagic {
		return false
	}

	itemInStashNotMatchingRule := false

	// Check if we already have the item in our stash and if it doesn't match any of our pickit rules
	for _, it := range ctx.Data.Inventory.ByLocation(item.LocationStash, item.LocationSharedStash) {
		if it.Name == i.Name {
			_, res := ctx.CharacterCfg.Runtime.Rules.EvaluateAll(it)
			if res != nip.RuleResultFullMatch {
				itemInStashNotMatchingRule = true
			}
		}
	}

	recipeMatch := false

	// Check if the item is part of a recipe and if that recipe is enabled
	for _, recipe := range Recipes {
		if slices.Contains(recipe.Items, string(i.Name)) && slices.Contains(ctx.CharacterCfg.CubeRecipes.EnabledRecipes, recipe.Name) {
			recipeMatch = true
			break
		}
	}

	if recipeMatch && !itemInStashNotMatchingRule {
		return true
	}

	return false
}

// Helper function to get skill name from layer
func getSkillNameFromLayer(statID stat.ID, layer int) string {
	skillAliases := map[stat.ID]map[int]string{
		stat.ID(83): {
			0: "Amazon Skills",
			1: "Sorceress Skills",
			2: "Necromancer Skills",
			3: "Paladin Skills",
			4: "Barbarian Skills",
			5: "Druid Skills",
			6: "Assassin Skills",
		},
		stat.ID(188): {
			0:  "Bow and Crossbow Skills",
			1:  "Passive and Magic Skills",
			2:  "Javelin and Spear Skills",
			8:  "Fire Skills",
			9:  "Lightning Skills",
			10: "Cold Skills",
			16: "Curses Skills",
			17: "Poison and Bone Skills",
			18: "Necromancer Summoning Skills",
			24: "Paladin Combat Skills",
			25: "Offensive Auras Skills",
			26: "Defensive Auras Skills",
			32: "Barbarian Combat Skills",
			33: "Masteries Skills",
			34: "Warcries Skills",
			40: "Druid Summoning Skills",
			41: "Shapeshifting Skills",
			42: "Elemental Skills",
			48: "Traps Skills",
			49: "Shadow Disciplines Skills",
			50: "Martial Arts Skills",
		},
	}

	if aliases, exists := skillAliases[statID]; exists {
		if name, hasAlias := aliases[layer]; hasAlias {
			return name
		}
	}
	return statID.String()
}

func formatItemStats(i data.Item) string {
	stats := []string{
		fmt.Sprintf("Name: %s", i.Name),
		fmt.Sprintf("Quality: %s", i.Quality.ToString()),
		fmt.Sprintf("Ethereal: %v", i.Ethereal),
	}

	resistStats := make(map[stat.ID]int)
	otherStats := []string{}

	formatStat := func(statData stat.Data) string {
		if statData.ID == stat.Durability || statData.ID == stat.MaxDurability {
			return ""
		}

		if isResistStat(statData.ID) {
			resistStats[statData.ID] = statData.Value
			return ""
		}

		// Spezielle Behandlung für Skill-Stats
		if statData.ID == stat.ID(83) || statData.ID == stat.ID(188) { // 83=ItemAddClassSkill, 188=ItemAddSkillTab
			skillName := getSkillNameFromLayer(statData.ID, statData.Layer)
			return fmt.Sprintf("+%d to %s", statData.Value, skillName)
		}

		// Normales Stat-Formatting für andere Stats
		if statInfo, exists := StatMap[statData.ID]; exists {
			if statInfo.UsePrefix && statData.Value > 0 {
				return fmt.Sprintf("+%d %s", statData.Value, statData.ID.String())
			}
			return fmt.Sprintf("%d %s", statData.Value, statData.ID.String())
		}
		return fmt.Sprintf("%d %s", statData.Value, statData.ID.String())
	}

	// Process item stats
	for _, statData := range i.Stats {
		formattedStat := formatStat(statData)
		if formattedStat != "" {
			otherStats = append(otherStats, formattedStat)
		}
	}

	if len(otherStats) > 0 {
		stats = append(stats, otherStats...)
	}

	// Handle resistances
	resistIDs := []stat.ID{stat.FireResist, stat.LightningResist, stat.ColdResist, stat.PoisonResist}
	if hasAllResistances(resistStats, resistIDs) && allValuesEqual(resistStats, resistIDs) {
		stats = append(stats, fmt.Sprintf("+%d All Res", resistStats[stat.FireResist]))
	} else {
		for _, id := range resistIDs {
			if val, exists := resistStats[id]; exists {
				stats = append(stats, fmt.Sprintf("+%d %s", val, id.String()))
			}
		}
	}

	return strings.Join(stats, "\n")
}

// Helper function to check if a stat is a resistance stat
func isResistStat(id stat.ID) bool {
	return id == stat.FireResist ||
		id == stat.LightningResist ||
		id == stat.ColdResist ||
		id == stat.PoisonResist
}

// Helper function to check if all resistances are present
func hasAllResistances(statsMap map[stat.ID]int, ids []stat.ID) bool {
	for _, id := range ids {
		if _, exists := statsMap[id]; !exists {
			return false
		}
	}
	return true
}

// Helper function to check if all values are equal
func allValuesEqual(statsMap map[stat.ID]int, ids []stat.ID) bool {
	if len(ids) == 0 {
		return false
	}
	firstVal := statsMap[ids[0]]
	for _, id := range ids[1:] {
		if statsMap[id] != firstVal {
			return false
		}
	}
	return true
}
func stashItemAction(i data.Item, rule string, ruleFile string, firstRun bool) bool {
	ctx := context.Get()
	ctx.SetLastAction("stashItemAction")
	screenPos := ui.GetScreenCoordsForItem(i)
	ctx.HID.MovePointer(screenPos.X, screenPos.Y)
	utils.Sleep(170)
	utils.Sleep(150)
	ctx.HID.ClickWithModifier(game.LeftButton, screenPos.X, screenPos.Y, game.CtrlKey)
	utils.Sleep(500)

	// Check if item was successfully moved
	for _, it := range ctx.Data.Inventory.ByLocation(item.LocationInventory) {
		if it.UnitID == i.UnitID {
			return false
		}
	}

	// Don't log items that we already have in inventory during first run
	if !firstRun {
		var baseEvent event.BaseEvent
		if i.Quality == 0x02 {
			// Simple string conversion instead of Sprintf
			message := string(i.Name)
			baseEvent = event.WithoutScreenshot(ctx.Name, message)
		} else {
			// For items that would have had screenshots, include detailed stats
			statsMsg := formatItemStats(i)
			message := fmt.Sprintf("**%s:**\n%s", ctx.Name, statsMsg)
			baseEvent = event.WithoutScreenshot(ctx.Name, message)
		}

		event.Send(event.ItemStashed(baseEvent, data.Drop{
			Item:     i,
			Rule:     rule,
			RuleFile: ruleFile,
		}))
	}
	return true
}

func clickStashGoldBtn() {
	ctx := context.Get()
	ctx.SetLastStep("clickStashGoldBtn")

	utils.Sleep(170)
	if ctx.GameReader.LegacyGraphics() {
		ctx.HID.Click(game.LeftButton, ui.StashGoldBtnXClassic, ui.StashGoldBtnYClassic)
		utils.Sleep(1000)
		ctx.HID.Click(game.LeftButton, ui.StashGoldBtnConfirmXClassic, ui.StashGoldBtnConfirmYClassic)
	} else {
		ctx.HID.Click(game.LeftButton, ui.StashGoldBtnX, ui.StashGoldBtnY)
		utils.Sleep(1000)
		ctx.HID.Click(game.LeftButton, ui.StashGoldBtnConfirmX, ui.StashGoldBtnConfirmY)
	}
}

func SwitchStashTab(tab int) {
	ctx := context.Get()
	ctx.SetLastStep("switchTab")

	if ctx.GameReader.LegacyGraphics() {
		x := ui.SwitchStashTabBtnXClassic
		y := ui.SwitchStashTabBtnYClassic

		tabSize := ui.SwitchStashTabBtnTabSizeClassic
		x = x + tabSize*tab - tabSize/2
		ctx.HID.Click(game.LeftButton, x, y)
		utils.Sleep(500)
	} else {
		x := ui.SwitchStashTabBtnX
		y := ui.SwitchStashTabBtnY

		tabSize := ui.SwitchStashTabBtnTabSize
		x = x + tabSize*tab - tabSize/2
		ctx.HID.Click(game.LeftButton, x, y)
		utils.Sleep(500)
	}
}

func OpenStash() error {
	ctx := context.Get()
	ctx.SetLastAction("OpenStash")

	bank, found := ctx.Data.Objects.FindOne(object.Bank)
	if !found {
		return errors.New("stash not found")
	}
	InteractObject(bank,
		func() bool {
			return ctx.Data.OpenMenus.Stash
		},
	)

	return nil
}

func CloseStash() error {
	ctx := context.Get()
	ctx.SetLastAction("CloseStash")

	if ctx.Data.OpenMenus.Stash {
		ctx.HID.PressKey(win.VK_ESCAPE)
	} else {
		return errors.New("stash is not open")
	}

	return nil
}

func TakeItemsFromStash(stashedItems []data.Item) error {
	ctx := context.Get()
	ctx.SetLastAction("TakeItemsFromStash")

	if ctx.Data.OpenMenus.Stash {
		err := OpenStash()
		if err != nil {
			return err
		}
	}

	utils.Sleep(250)

	for _, i := range stashedItems {

		if i.Location.LocationType != item.LocationStash && i.Location.LocationType != item.LocationSharedStash {
			continue
		}

		// Make sure we're on the correct tab
		SwitchStashTab(i.Location.Page + 1)

		// Move the item to the inventory
		screenPos := ui.GetScreenCoordsForItem(i)
		ctx.HID.MovePointer(screenPos.X, screenPos.Y)
		ctx.HID.ClickWithModifier(game.LeftButton, screenPos.X, screenPos.Y, game.CtrlKey)
		utils.Sleep(500)
	}

	return nil
}
