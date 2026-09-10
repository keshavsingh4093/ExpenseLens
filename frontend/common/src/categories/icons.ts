import {
  Utensils, ShoppingBasket, Fuel, ShoppingBag, HeartPulse, Clapperboard, Plane, PlugZap, CircleDashed, Tag,
  Coffee, Car, House, Gift, Book, Dumbbell, Baby, PawPrint, Shirt, Smartphone, Wifi, Bus, Pill, GraduationCap,
  Music, Gamepad2, Wrench, Briefcase, CreditCard, Sparkles,
  type LucideIcon,
} from 'lucide-react'

/** Icon names stored on categories (system seeds + picker choices). */
export const categoryIcons: Record<string, LucideIcon> = {
  utensils: Utensils, 'shopping-basket': ShoppingBasket, fuel: Fuel, 'shopping-bag': ShoppingBag,
  'heart-pulse': HeartPulse, clapperboard: Clapperboard, plane: Plane, 'plug-zap': PlugZap, 'circle-dashed': CircleDashed, tag: Tag,
  coffee: Coffee, car: Car, house: House, gift: Gift, book: Book, dumbbell: Dumbbell, baby: Baby, 'paw-print': PawPrint,
  shirt: Shirt, smartphone: Smartphone, wifi: Wifi, bus: Bus, pill: Pill, 'graduation-cap': GraduationCap, music: Music,
  'gamepad-2': Gamepad2, wrench: Wrench, briefcase: Briefcase, 'credit-card': CreditCard, sparkles: Sparkles,
}

export const categoryIconNames = Object.keys(categoryIcons)
