import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.onnx
import os
import sys

# ... Check for GPU ...
device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
print(f"Using device: {device}")

# ==========================================
# Embedded RRDBNet Architecture
# ==========================================

def make_layer(basic_block, num_basic_block, **kwarg):
    """Make layers by stacking the same blocks.
    Args:
        basic_block (nn.module): nn.module class for basic block.
        num_basic_block (int): number of blocks.
    Returns:
        nn.Sequential: Stacked blocks in nn.Sequential.
    """
    layers = []
    for _ in range(num_basic_block):
        layers.append(basic_block(**kwarg))
    return nn.Sequential(*layers)

class ResidualDenseBlock(nn.Module):
    """Residual Dense Block.

    Used in RRDB block in ESRGAN.

    Args:
        num_feat (int): Channel number of intermediate features.
        num_grow_ch (int): Channels for each growth.
    """

    def __init__(self, num_feat=64, num_grow_ch=32):
        super(ResidualDenseBlock, self).__init__()
        self.conv1 = nn.Conv2d(num_feat, num_grow_ch, 3, 1, 1)
        self.conv2 = nn.Conv2d(num_feat + num_grow_ch, num_grow_ch, 3, 1, 1)
        self.conv3 = nn.Conv2d(num_feat + 2 * num_grow_ch, num_grow_ch, 3, 1, 1)
        self.conv4 = nn.Conv2d(num_feat + 3 * num_grow_ch, num_grow_ch, 3, 1, 1)
        self.conv5 = nn.Conv2d(num_feat + 4 * num_grow_ch, num_feat, 3, 1, 1)

        self.lrelu = nn.LeakyReLU(negative_slope=0.2, inplace=True)

        # initialization
        # default_init_weights([self.conv1, self.conv2, self.conv3, self.conv4, self.conv5], 0.1)

    def forward(self, x):
        x1 = self.lrelu(self.conv1(x))
        x2 = self.lrelu(self.conv2(torch.cat((x, x1), 1)))
        x3 = self.lrelu(self.conv3(torch.cat((x, x1, x2), 1)))
        x4 = self.lrelu(self.conv4(torch.cat((x, x1, x2, x3), 1)))
        x5 = self.conv5(torch.cat((x, x1, x2, x3, x4), 1))
        # Emperically, we use 0.2 to scale the residual for better performance
        return x5 * 0.2 + x


class RRDB(nn.Module):
    """Residual in Residual Dense Block.

    Used in RRDB block in ESRGAN.

    Args:
        num_feat (int): Channel number of intermediate features.
        num_grow_ch (int): Channels for each growth.
    """

    def __init__(self, num_feat, num_grow_ch=32):
        super(RRDB, self).__init__()
        self.rdb1 = ResidualDenseBlock(num_feat, num_grow_ch)
        self.rdb2 = ResidualDenseBlock(num_feat, num_grow_ch)
        self.rdb3 = ResidualDenseBlock(num_feat, num_grow_ch)

    def forward(self, x):
        out = self.rdb1(x)
        out = self.rdb2(out)
        out = self.rdb3(out)
        # Emperically, we use 0.2 to scale the residual for better performance
        return out * 0.2 + x


class RRDBNet(nn.Module):
    """Networks consisting of Residual in Residual Dense Block, which is used in ESRGAN.

    ESRGAN: Enhanced Super-Resolution Generative Adversarial Networks.
    """

    def __init__(self, num_in_ch, num_out_ch, scale=4, num_feat=64, num_block=23, num_grow_ch=32):
        super(RRDBNet, self).__init__()
        self.scale = scale
        self.conv_first = nn.Conv2d(num_in_ch, num_feat, 3, 1, 1)
        self.RRDB_trunk = make_layer(RRDB, num_block, num_feat=num_feat, num_grow_ch=num_grow_ch)
        self.trunk_conv = nn.Conv2d(num_feat, num_feat, 3, 1, 1)
        #### upsampling
        self.upconv1 = nn.Conv2d(num_feat, num_feat, 3, 1, 1)
        self.upconv2 = nn.Conv2d(num_feat, num_feat, 3, 1, 1)
        self.HR_conv = nn.Conv2d(num_feat, num_feat, 3, 1, 1)
        self.conv_last = nn.Conv2d(num_feat, num_out_ch, 3, 1, 1)

        self.lrelu = nn.LeakyReLU(negative_slope=0.2, inplace=True)

    def forward(self, x):
        feat = self.conv_first(x)
        body_feat = self.trunk_conv(self.RRDB_trunk(feat))
        feat = feat + body_feat
        feat = self.lrelu(self.upconv1(F.interpolate(feat, scale_factor=2, mode='nearest')))
        feat = self.lrelu(self.upconv2(F.interpolate(feat, scale_factor=2, mode='nearest')))
        out = self.conv_last(self.lrelu(self.HR_conv(feat)))
        return out

# ==========================================
# Conversion Logic
# ==========================================

def get_model_config(model_name):
    if model_name == 'realesrgan-x4plus':
        return {
            'url': 'https://github.com/xinntao/Real-ESRGAN/releases/download/v0.1.0/RealESRGAN_x4plus.pth',
            'net_scale': 4,
            'num_in_ch': 3,
            'num_out_ch': 3,
            'num_block': 23,
            'num_feat': 64
        }
    elif model_name == 'realesrgan-x4plus-anime':
        return {
            'url': 'https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.2.4/RealESRGAN_x4plus_anime_6B.pth',
            'net_scale': 4,
            'num_in_ch': 3,
            'num_out_ch': 3,
            'num_block': 6, # Note: Anime model is smaller (6 blocks)
            'num_feat': 64
        }
    else:
        raise ValueError(f"Unknown model: {model_name}")

def download_model(url, save_path):
    import requests
    if os.path.exists(save_path):
        print(f"Model already exists at {save_path}")
        return
    
    print(f"Downloading model from {url}...")
    try:
        response = requests.get(url, stream=True)
        response.raise_for_status()
        with open(save_path, 'wb') as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
        print("Download complete.")
    except Exception as e:
        print(f"Failed to download model: {e}")
        # Remove partial file
        if os.path.exists(save_path):
            os.remove(save_path)
        sys.exit(1)

def convert_to_onnx(model_name, output_dir):
    config = get_model_config(model_name)
    model_path = os.path.join(output_dir, f"{model_name}.pth")
    onnx_path = os.path.join(output_dir, f"{model_name}.onnx")
    
    # Download if not exists
    download_model(config['url'], model_path)
    
    if os.path.exists(onnx_path):
        print(f"ONNX model already exists at {onnx_path}, skipping conversion.")
        return

    print(f"Exporting {model_name} to ONNX...")
    
    model = RRDBNet(
        num_in_ch=config['num_in_ch'],
        num_out_ch=config['num_out_ch'],
        scale=config['net_scale'],
        num_block=config['num_block'],
        num_feat=config['num_feat']
    )
    
    try:
        loadnet = torch.load(model_path, map_location=device)
        
        # Handle state dict keys if needed
        if 'params_ema' in loadnet:
            keyname = 'params_ema'
        else:
            keyname = 'params'
            
        state_dict = loadnet[keyname]
        
        # fix keys
        new_state_dict = {}
        for k, v in state_dict.items():
            if k.startswith('body.'):
                # Map 'body' to 'RRDB_trunk'
                new_k = k.replace('body.', 'RRDB_trunk.')
                new_state_dict[new_k] = v
            elif k.startswith('conv_body.'):
                new_k = k.replace('conv_body.', 'trunk_conv.')
                new_state_dict[new_k] = v
            elif k.startswith('conv_up1.'):
                new_k = k.replace('conv_up1.', 'upconv1.')
                new_state_dict[new_k] = v
            elif k.startswith('conv_up2.'):
                new_k = k.replace('conv_up2.', 'upconv2.')
                new_state_dict[new_k] = v
            elif k.startswith('conv_hr.'):
                new_k = k.replace('conv_hr.', 'HR_conv.')
                new_state_dict[new_k] = v
            else:
                new_state_dict[k] = v
                
        model.load_state_dict(new_state_dict)
    except Exception as e:
        print(f"Failed to load weights: {e}")
        return

    model.eval()
    model.to(device)
    
    # Create dummy input
    dummy_input = torch.randn(1, 3, 64, 64).to(device)
    
    # Export
    try:
        torch.onnx.export(
            model,
            dummy_input,
            onnx_path,
            opset_version=14, # Real-ESRGAN recommends opset 11 or higher
            do_constant_folding=True,
            input_names=['input'],
            output_names=['output'],
            dynamic_axes={
                'input': {0: 'batch_size', 2: 'height', 3: 'width'},
                'output': {0: 'batch_size', 2: 'height', 3: 'width'}
            }
        )
        print(f"Successfully exported to {onnx_path}")
    except Exception as e:
        print(f"Export failed: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python convert.py <output_dir>")
        sys.exit(1)
        
    output_dir = sys.argv[1]
    os.makedirs(output_dir, exist_ok=True)
    
    convert_to_onnx('realesrgan-x4plus', output_dir)
    convert_to_onnx('realesrgan-x4plus-anime', output_dir)
