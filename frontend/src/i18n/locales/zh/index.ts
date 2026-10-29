import adminAccessKeys from './adminAccessKeys'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import canvas from './canvas'
import admin from './admin'
import misc from './misc'
import fork from './fork'
import cheapRouter from './cheapRouter'
import { deepMerge } from '../../deepMerge'

export default deepMerge(deepMerge({
  adminAccessKeys,
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  ...canvas,
  admin,
  ...misc,
}, fork), cheapRouter)
